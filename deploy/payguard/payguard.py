#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""sub2api 支付安全巡检 / 订单对账

用法:
  payguard.py guard        # 巡检: 秒到账订单 / 陌生IP回调 / 回调验签失败突增 / 新号大额充值
  payguard.py reconcile    # 对账: 最近订单逐笔向支付网关核实
  payguard.py selftest     # 只跑检查不写告警状态
  payguard.py test-notify  # 发一条测试告警，验证邮件/微信通道

退出码: 0 正常, 1 发现告警
"""

import json
import os
import re
import smtplib
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from email.header import Header
from email.mime.text import MIMEText
from email.utils import formataddr

BASE = "/data/sub2api/security"
LOG = "/data/sub2api/data/logs/payguard.log"
STATE = os.path.join(BASE, "state.json")
CONF = os.path.join(BASE, "config.json")

DEFAULT_CONF = {
    # 真实支付网关回调来源 IP（从历史日志提取，换网关/换IP时需更新）
    "gateway_ips": [
        "159.75.222.226",
        "123.207.78.140",
        "203.195.200.21",
        "118.31.74.116",
        "134.175.49.240",
    ],
    # 告警邮件（SMTP 复用站点 settings 里的 smtp_* 配置，这里只填收件人）
    "mail": {
        "enabled": True,
        "to": ["recurdream@163.com"],
    },
    # 微信推送: type = wecom / serverchan / pushplus
    #   wecom      -> token 填企业微信群机器人完整 webhook 地址
    #   serverchan -> token 填 sct.ftqq.com 的 SendKey（SCT 开头）
    #   pushplus   -> token 填 pushplus.plus 的 token（32 位）
    "wechat": {
        "type": "",
        "token": "",
    },
    # 可选: 通用 webhook（企业微信/钉钉/Telegram 自定义接口）
    "notify_url": "",
    # 下单到支付间隔小于该值视为异常（真人扫码不可能这么快）
    "fast_paid_seconds": 2,
    # 新注册多少小时内的大额充值要告警
    "new_user_hours": 24,
    "new_user_max_amount": 200,
    # 对账回看多少小时
    "reconcile_hours": 3,
    # 异常订单回看多少天（用户量不大，给足窗口，重跑也不会漏）
    "lookback_days": 30,
    # 15 分钟内验签失败超过该次数即告警（疑似扫描）
    "verify_fail_threshold": 10,
}

PSQL = [
    "docker", "exec", "-i", "sub2api-postgres",
    "psql", "-U", "sub2api", "-d", "sub2api",
    "-t", "-A", "-F", "|", "-c",
]


def load_conf():
    conf = dict(DEFAULT_CONF)
    try:
        with open(CONF) as f:
            conf.update(json.load(f))
    except Exception:
        pass
    return conf


def load_state():
    try:
        with open(STATE) as f:
            return json.load(f)
    except Exception:
        return {"alerted_orders": [], "alerted_ips": [], "last_run": 0}


def save_state(state):
    tmp = STATE + ".tmp"
    with open(tmp, "w") as f:
        json.dump(state, f)
    os.replace(tmp, STATE)


def write_log(level, msg):
    line = "%s [%s] %s" % (time.strftime("%Y-%m-%d %H:%M:%S"), level, msg)
    print(line)
    try:
        with open(LOG, "a") as f:
            f.write(line + "\n")
    except Exception:
        pass


def get_smtp_config():
    """复用站点 settings 里的 smtp_* 配置"""
    rows = psql("SELECT key, value FROM settings WHERE key LIKE 'smtp%'")
    return dict((r[0], r[1]) for r in rows if len(r) >= 2)


def send_mail(subject, body, conf):
    mc = conf.get("mail") or {}
    to_list = [x.strip() for x in (mc.get("to") or []) if x and x.strip()]
    if not mc.get("enabled") or not to_list:
        return
    smtp = get_smtp_config()
    host = (smtp.get("smtp_host") or "").strip()
    user = (smtp.get("smtp_username") or "").strip()
    pwd = smtp.get("smtp_password") or ""
    sender = (smtp.get("smtp_from") or user).strip()
    name = (smtp.get("smtp_from_name") or "sub2api").strip()
    try:
        port = int(smtp.get("smtp_port") or 587)
    except ValueError:
        port = 587
    use_tls = (smtp.get("smtp_use_tls") or "").lower() == "true"
    if not host or not user or not pwd:
        write_log("WARN", "告警邮件未发送: 站点 SMTP 配置不完整")
        return
    msg = MIMEText(body, "plain", "utf-8")
    msg["Subject"] = Header(subject, "utf-8")
    msg["From"] = formataddr((str(Header(name, "utf-8")), sender))
    msg["To"] = ", ".join(to_list)
    # 阿里云默认封 25/587，优先走 465 SSL 直连，失败再回落配置端口
    if port == 465:
        candidates = [(465, True)]
    else:
        candidates = [(465, True), (port, False)]
    last_err = None
    for p, use_ssl in candidates:
        try:
            if use_ssl:
                s = smtplib.SMTP_SSL(host, p, timeout=20)
            else:
                s = smtplib.SMTP(host, p, timeout=20)
                s.ehlo()
                if use_tls:
                    s.starttls()
                    s.ehlo()
            s.login(user, pwd)
            s.sendmail(sender, to_list, msg.as_string())
            s.quit()
            write_log("INFO", "告警邮件已发送(%d) -> %s" % (p, ", ".join(to_list)))
            return
        except Exception as e:
            last_err = e
            continue
    write_log("WARN", "告警邮件发送失败: %s" % last_err)


def send_wechat(subject, body, conf):
    wx = conf.get("wechat") or {}
    typ = (wx.get("type") or "").strip().lower()
    token = (wx.get("token") or "").strip()
    if not typ or not token:
        return
    text = "### %s\n\n%s" % (subject, body.replace("\n", "\n\n"))
    try:
        if typ == "wecom":
            payload = json.dumps(
                {"msgtype": "markdown", "markdown": {"content": text}},
                ensure_ascii=False).encode("utf-8")
            req = urllib.request.Request(token, data=payload,
                                         headers={"Content-Type": "application/json"})
        elif typ == "serverchan":
            req = urllib.request.Request(
                "https://sctapi.ftqq.com/%s.send" % token,
                data=urllib.parse.urlencode({"title": subject, "desp": text}).encode("utf-8"))
        elif typ == "pushplus":
            req = urllib.request.Request(
                "https://www.pushplus.plus/send",
                data=urllib.parse.urlencode({
                    "token": token, "title": subject,
                    "content": text, "template": "markdown",
                }).encode("utf-8"))
        else:
            write_log("WARN", "未知微信通道类型: %s" % typ)
            return
        raw = urllib.request.urlopen(req, timeout=15).read().decode("utf-8", "replace")
        write_log("INFO", "微信告警已推送(%s): %s" % (typ, raw[:120]))
    except Exception as e:
        write_log("WARN", "微信告警发送失败(%s): %s" % (typ, e))


def notify(title, body, conf):
    send_mail(title, body, conf)
    send_wechat(title, body, conf)
    url = (conf.get("notify_url") or "").strip()
    if not url:
        return
    payload = json.dumps({"title": title, "content": body}, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=payload, headers={"Content-Type": "application/json"})
    try:
        urllib.request.urlopen(req, timeout=10).read()
    except Exception as e:
        write_log("WARN", "notify webhook failed: %s" % e)


def psql(sql, timeout=30):
    """执行 SQL，返回 list[list[str]]"""
    try:
        p = subprocess.run(PSQL + [sql], stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                           universal_newlines=True, timeout=timeout)
    except Exception as e:
        write_log("ERROR", "psql exception: %s" % e)
        return []
    if p.returncode != 0:
        write_log("ERROR", "psql failed: %s" % (p.stderr or "")[:300])
        return []
    rows = []
    for line in (p.stdout or "").splitlines():
        line = line.rstrip("\n")
        if line == "":
            continue
        rows.append(line.split("|"))
    return rows


def docker_logs(container, since, timeout=60):
    try:
        p = subprocess.run(["docker", "logs", container, "--since", since],
                           stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                           universal_newlines=True, timeout=timeout)
        return (p.stdout or "").splitlines()
    except Exception as e:
        write_log("ERROR", "docker logs failed: %s" % e)
        return []


IP_RE = re.compile(r'([0-9]{1,3}(?:\.[0-9]{1,3}){3})"\s*$')


def check_fast_paid(conf, state, alerts):
    secs = int(conf.get("fast_paid_seconds", 2))
    days = int(conf.get("lookback_days", 30))
    sql = (
        "SELECT o.id, o.user_email, o.amount, "
        "round(EXTRACT(EPOCH FROM (o.paid_at - o.created_at))::numeric, 3), "
        "coalesce(o.payment_trade_no,''), coalesce(o.client_ip,''), o.created_at "
        "FROM payment_orders o "
        "WHERE o.status IN ('COMPLETED','PAID','RECHARGING') AND o.paid_at IS NOT NULL "
        "AND o.paid_at - o.created_at < interval '%d seconds' "
        "AND o.created_at > now() - interval '%d days' "
        "ORDER BY o.created_at DESC" % (secs, days)
    )
    for r in psql(sql):
        oid = r[0]
        key = "fast:%s" % oid
        if key in state["alerted_orders"]:
            continue
        state["alerted_orders"].append(key)
        alerts.append(
            "秒到账订单 order_id=%s 用户=%s 金额=%s 间隔=%ss 交易号=%s IP=%s 时间=%s"
            % (oid, r[1], r[2], r[3], r[4], r[5], r[6])
        )


def check_webhook_ips(conf, state, alerts):
    allow = set(conf.get("gateway_ips") or [])
    lines = docker_logs("sub2api-nginx", "20m")
    hit = {}
    for line in lines:
        if "webhook/" not in line:
            continue
        m = IP_RE.search(line)
        if not m:
            continue
        ip = m.group(1)
        if ip in allow:
            continue
        # 内部地址不算
        if ip.startswith("127.") or ip.startswith("172.") or ip.startswith("10.") or ip.startswith("192.168."):
            continue
        hit[ip] = hit.get(ip, 0) + 1
    for ip, cnt in sorted(hit.items(), key=lambda x: -x[1]):
        key = "ip:%s:%s" % (ip, time.strftime("%Y-%m-%d"))
        if key in state["alerted_ips"]:
            continue
        state["alerted_ips"].append(key)
        alerts.append("陌生 IP 回调支付网关 IP=%s 次数=%d（非白名单，可能是伪造/扫描）" % (ip, cnt))


def check_verify_fail_burst(conf, state, alerts):
    threshold = int(conf.get("verify_fail_threshold", 10))
    cnt = 0
    for line in docker_logs("sub2api-nginx", "15m"):
        if "webhook/" in line and re.search(r'" 40[0-9] ', line):
            cnt += 1
    if cnt >= threshold:
        bucket = time.strftime("%Y-%m-%d-%H")
        key = "vfail:%s" % bucket
        if key not in state["alerted_ips"]:
            state["alerted_ips"].append(key)
            alerts.append("15 分钟内支付回调验签失败 %d 次（疑似伪造回调扫描）" % cnt)


def check_new_user_big(conf, state, alerts):
    hours = int(conf.get("new_user_hours", 24))
    amount = float(conf.get("new_user_max_amount", 200))
    days = int(conf.get("lookback_days", 30))
    sql = (
        "SELECT o.id, o.user_email, o.amount, o.created_at, u.created_at "
        "FROM payment_orders o JOIN users u ON u.id = o.user_id "
        "WHERE o.status IN ('COMPLETED','PAID','RECHARGING') "
        "AND o.created_at - u.created_at < interval '%d hours' "
        "AND o.amount >= %s AND o.created_at > now() - interval '%d days' "
        "ORDER BY o.created_at DESC" % (hours, amount, days)
    )
    for r in psql(sql):
        key = "newuser:%s" % r[0]
        if key in state["alerted_orders"]:
            continue
        state["alerted_orders"].append(key)
        alerts.append(
            "新号大额充值 order_id=%s 用户=%s 金额=%s 下单=%s 注册=%s"
            % (r[0], r[1], r[2], r[3], r[4])
        )


def do_guard(conf, state, write=True):
    alerts = []
    check_fast_paid(conf, state, alerts)
    check_webhook_ips(conf, state, alerts)
    check_verify_fail_burst(conf, state, alerts)
    check_new_user_big(conf, state, alerts)
    if alerts:
        for a in alerts:
            write_log("ALERT", a)
        notify("支付安全告警", "\n".join(alerts), conf)
        if write:
            save_state(state)
        return 1
    write_log("OK", "巡检通过")
    if write:
        save_state(state)
    return 0


def get_active_gateway():
    """取当前启用的 easypay 实例配置，返回 (instance_id, config)"""
    sql = (
        "SELECT id, config::text FROM payment_provider_instances "
        "WHERE provider_key = 'easypay' AND enabled = true ORDER BY id LIMIT 1"
    )
    rows = psql(sql)
    if not rows:
        return None, None
    try:
        inst_id = int(rows[0][0])
        cfg = json.loads(rows[0][1])
    except Exception:
        return None, None
    if not cfg.get("apiBase") or not cfg.get("pid") or not cfg.get("pkey"):
        return None, None
    return inst_id, cfg


def query_upstream(cfg, out_trade_no):
    """向网关查单，返回 dict 或 None"""
    import urllib.parse
    base = cfg["apiBase"].rstrip("/")
    qs = urllib.parse.urlencode({
        "act": "order",
        "pid": cfg["pid"],
        "key": cfg["pkey"],
        "out_trade_no": out_trade_no,
    })
    url = base + "/api.php?" + qs
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "sub2api-payguard/1.0"})
        raw = urllib.request.urlopen(req, timeout=15).read().decode("utf-8", "replace")
    except Exception as e:
        return {"error": str(e)[:200]}
    try:
        data = json.loads(raw)
    except Exception:
        return {"error": "non-json response: " + raw[:120]}
    node = data.get("data") if isinstance(data.get("data"), dict) else data
    status = node.get("trade_status") or node.get("status")
    money = node.get("money")
    return {
        "status": str(status) if status is not None else "",
        "money": str(money) if money is not None else "",
        "trade_no": str(node.get("trade_no") or ""),
        "raw_code": data.get("code"),
        "raw_msg": data.get("msg"),
    }


def do_reconcile(conf, state, write=True):
    inst_id, cfg = get_active_gateway()
    if not cfg:
        write_log("WARN", "对账跳过: 未找到启用的 easypay 实例")
        return 0
    hours = int(conf.get("reconcile_hours", 3))
    sql = (
        "SELECT id, out_trade_no, user_email, amount::text, "
        "coalesce(payment_trade_no,''), status "
        "FROM payment_orders "
        "WHERE status IN ('COMPLETED','PAID') AND payment_type = 'alipay' "
        "AND provider_instance_id = '%s' "
        "AND created_at > now() - interval '%d hours' "
        "ORDER BY created_at DESC" % (inst_id, hours)
    )
    rows = psql(sql)
    if not rows:
        write_log("OK", "对账: 近 %d 小时无可对账订单" % hours)
        return 0

    alerts = []
    checked = 0
    for r in rows:
        oid, out_trade_no, email, amount, trade_no, status = r[0], r[1], r[2], r[3], r[4], r[5]
        res = query_upstream(cfg, out_trade_no)
        checked += 1
        if not res or "error" in res:
            # 网关查询失败不直接判为异常（可能超时），记为提示
            write_log("INFO", "对账跳过 order=%s: %s" % (oid, (res or {}).get("error", "no data")))
            continue
        st = (res.get("status") or "").lower()
        is_paid = st in ("1", "true", "paid", "trade_success", "trade_finished")
        st_upper = st.replace("trade_", "").upper()
        if st_upper in ("SUCCESS", "FINISHED"):
            is_paid = True
        key = "recon:%s" % oid
        if not is_paid:
            if key not in state["alerted_orders"]:
                state["alerted_orders"].append(key)
                alerts.append(
                    "对账不一致 order=%s 用户=%s 金额=%s 本地状态=%s 网关状态=%s msg=%s"
                    % (oid, email, amount, status, res.get("status"), res.get("raw_msg"))
                )
            continue
        try:
            gw_amount = float(res.get("money") or 0)
            local_amount = float(amount)
            if abs(gw_amount - local_amount) > 0.01:
                if key not in state["alerted_orders"]:
                    state["alerted_orders"].append(key)
                    alerts.append("对账金额不符 order=%s 本地=%s 网关=%s" % (oid, local_amount, gw_amount))
        except Exception:
            pass
        if res.get("trade_no") and trade_no and res["trade_no"] != trade_no:
            if key not in state["alerted_orders"]:
                state["alerted_orders"].append(key)
                alerts.append("对账交易号不符 order=%s 本地=%s 网关=%s" % (oid, trade_no, res["trade_no"]))

    if alerts:
        for a in alerts:
            write_log("ALERT", a)
        notify("支付对账异常", "\n".join(alerts), conf)
        if write:
            save_state(state)
        return 1
    write_log("OK", "对账通过，共核对 %d 笔" % checked)
    if write:
        save_state(state)
    return 0


def main():
    modes = ("guard", "reconcile", "selftest", "test-notify")
    if len(sys.argv) < 2 or sys.argv[1] not in modes:
        print(__doc__)
        return 2
    conf = load_conf()
    state = load_state()
    mode = sys.argv[1]
    if mode == "guard":
        code = do_guard(conf, state)
    elif mode == "reconcile":
        code = do_reconcile(conf, state)
    elif mode == "test-notify":
        body = ("这是一条测试告警，收到说明通道正常。\n"
                "主机: %s\n时间: %s" % (os.uname()[1], time.strftime("%Y-%m-%d %H:%M:%S")))
        notify("【测试】sub2api 支付安全告警", body, conf)
        write_log("INFO", "test-notify 已触发")
        code = 0
    else:
        # selftest: 只打印检查结果，不落状态、不推送
        alerts = []
        check_fast_paid(conf, load_state(), alerts)
        check_webhook_ips(conf, load_state(), alerts)
        check_verify_fail_burst(conf, load_state(), alerts)
        check_new_user_big(conf, load_state(), alerts)
        if alerts:
            for a in alerts:
                print("[ALERT] " + a)
        else:
            print("selftest ok: no alert")
        code = 0
    state["last_run"] = int(time.time())
    return code


if __name__ == "__main__":
    sys.exit(main())
