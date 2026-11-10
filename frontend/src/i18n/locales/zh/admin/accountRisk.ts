export default {
  accountRisk: {
    title: '账号风控',
    description: '针对注册机、撞库与凭证盗用的服务端风控。命中结果写入操作日志（security.risk.*），管理员与免检用户只留痕不处置。',
    reload: '重新加载',
    save: '保存配置',
    saving: '保存中…',
    saved: '风控配置已保存，约 30 秒内生效',
    saveFailed: '保存风控配置失败',
    loadFailed: '加载风控配置失败',
    master: {
      title: '总开关',
      hint: '关闭后所有规则停止评估，登录防护同时失效。',
      on: '已启用',
      off: '已停用',
      rules: '已注册规则：{rules}'
    },
    fields: {
      action: '处置动作',
      windowHours: '统计窗口（小时）'
    },
    actions: {
      flag: '仅留痕',
      ban: '封禁账号',
      block: '拒绝注册'
    },
    probe: {
      title: '注册即探测',
      hint: '注册后一段时间内，若账号用脚本客户端（python / curl / 空 UA 等）调用接口，判定为注册机。',
      window: '观察窗口（分钟）'
    },
    multi: {
      title: '同 IP 批量注册',
      hint: '同一 IP 在窗口内注册超过上限即拦截；动作为「拒绝注册」时在建号前拦下。',
      max: '每 IP 上限'
    },
    session: {
      title: '会话指纹异常',
      hint: '同一账号反复出现 IP+UA 指纹漂移，通常是凭证被复制到别的环境使用。',
      max: '最大不匹配次数'
    },
    login: {
      title: '登录暴力破解防护',
      hint: '按 IP 与按账号两个维度统计失败次数，超限临时锁定，不封禁账号。',
      maxIp: '单 IP 失败上限',
      ipWindow: 'IP 窗口（分钟）',
      maxAccount: '单账号失败上限',
      accountWindow: '账号窗口（分钟）',
      lock: '锁定（分钟）'
    },
    events: {
      title: '命中记录',
      hint: '来自操作日志，按 action 前缀 security.risk. 过滤。',
      empty: '暂无命中记录',
      loadFailed: '加载命中记录失败',
      filters: {
        all: '全部事件',
        banned: '已封禁',
        blocked: '注册被拦',
        loginLocked: '登录锁定',
        flagged: '仅留痕'
      },
      actions: {
        banned: '已封禁',
        blocked: '注册被拦',
        loginLocked: '登录锁定',
        flagged: '仅留痕'
      },
      columns: {
        time: '时间',
        user: '用户',
        ip: '客户端 IP',
        action: '处置',
        rule: '规则',
        reason: '原因'
      }
    }
  }
}
