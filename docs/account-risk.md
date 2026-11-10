# 账号异常检测与自动封禁

面向注册机 / 撞库 / token 盗用的服务端风控框架。设计目标是**判定与处置解耦**：
新增一类异常只需要加一个事件、一段配置、一条规则，处置动作（留痕 / 封禁 / 拦截）
由框架统一执行。

- 代码入口：`backend/internal/service/account_risk.go`
- 规则实现：`backend/internal/service/account_risk_rules.go`
- 登录防护：`backend/internal/service/account_risk_login.go`
- 配置读写：`backend/internal/service/account_risk_config.go`

## 1. 当前内置规则

| 规则名 | 触发事件 | 判定 | 默认动作 |
| --- | --- | --- | --- |
| `register_probe` | `first_api` | 注册后 120 分钟内的鉴权请求使用了非浏览器客户端（脚本 UA / 空 UA） | `ban` |
| `multi_account_ip` | `register` | 同一 IP 在 24 小时内成功注册超过 3 个账号 | `block`（建号前拦截，429） |
| `session_abuse` | `session_mismatch` | 同一账号 24 小时内 IP+UA 指纹不匹配超过 12 次 | `flag`（只留痕） |
| `login_bruteforce` | 登录失败 | 单 IP 10 分钟 30 次失败 / 单账号 15 分钟 8 次失败 | 临时锁定 15 分钟 |

> `register_probe` 的三档判定：
> 明确脚本 UA（`python-requests` / `curl` / `Go-http-client` / `okhttp` …）或空 UA → 按配置动作处置；
> 无法识别的非浏览器 UA（自研客户端）→ 只 `flag`，避免误杀；
> 浏览器 UA → 直接忽略。

## 2. 处置动作

| 动作 | 含义 |
| --- | --- |
| `flag` | 只写审计事件，不动账号（观察期 / 低置信度） |
| `ban` | `users.status = disabled` + 失效认证缓存，立即生效，管理员可后台恢复 |
| `block` | 拒绝本次请求（仅注册类规则使用，返回 429 `REGISTRATION_BLOCKED_RISK`） |

管理员（`role=admin`）与 `allowlisted_user_ids` 中的账号永远只留痕、不处置。

## 3. 审计留痕

所有命中都写入管理面审计日志（`/admin/audit-logs`），动作名：

- `security.risk.triggered` — 命中但只留痕
- `security.risk.account_banned` — 已自动封禁
- `security.risk.registration_blocked` — 注册被拦截
- `security.risk.login_locked` — 登录触发临时锁定

`extra` 字段包含命中的规则名、计数、窗口、阈值、UA、路径，可直接用于复盘误判。

## 4. 配置

存储在 `settings.account_risk_config`（JSON 文本），进程内缓存 30s。

```json
{
  "enabled": true,
  "register_probe": { "enabled": true, "window_minutes": 120, "action": "ban" },
  "multi_account": { "enabled": true, "max_accounts_per_ip": 3, "window_hours": 24, "action": "block" },
  "session_abuse": { "enabled": true, "max_mismatches": 12, "window_hours": 24, "action": "flag" },
  "login_guard": {
    "enabled": true,
    "max_ip_failures": 30,
    "ip_window_minutes": 10,
    "max_account_failures": 8,
    "account_window_minutes": 15,
    "lock_minutes": 15
  },
  "allowlisted_user_ids": []
}
```

修改示例（改完 30s 内生效，无需重启）：

```sql
UPDATE settings
SET value = jsonb_set(value::jsonb, '{register_probe,action}', '"flag"')::text
WHERE key = 'account_risk_config';
```

也可以走服务端接口 `SettingService.SetAccountRiskConfig`（会立即刷新缓存）。

### 应急：整体关闭

```sql
INSERT INTO settings (key, value, updated_at)
VALUES ('account_risk_config', '{"enabled":false}', now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
```

## 5. 扩展一类新的异常

以"新账号批量创建 API Key"为例：

1. 加事件（`account_risk.go`）：

   ```go
   const RiskEventAPIKeyCreate AccountRiskEventType = "api_key_create"
   ```

2. 加配置（`account_risk_config.go`）：

   ```go
   type AccountRiskAPIKeyBurstConfig struct {
       Enabled      bool       `json:"enabled"`
       MaxPerWindow int        `json:"max_per_window"`
       WindowHours  int        `json:"window_hours"`
       Action       RiskAction `json:"action"`
   }
   ```

   在 `AccountRiskConfig` 里挂上该字段，并在 `normalize()` 里补默认值与区间收敛。

3. 写规则（`account_risk_rules.go`）：

   ```go
   type apiKeyBurstRule struct{}
   func (apiKeyBurstRule) Name() string { return "api_key_burst" }
   func (apiKeyBurstRule) Events() []AccountRiskEventType {
       return []AccountRiskEventType{RiskEventAPIKeyCreate}
   }
   func (apiKeyBurstRule) Evaluate(ctx context.Context, eval *AccountRiskEvalContext) []AccountRiskVerdict {
       // 读 eval.Config / eval.Store / eval.User，返回 []AccountRiskVerdict
   }
   ```

4. 注册到 `defaultAccountRiskRules()`。

5. 在事件产生处上报：

   ```go
   service.AccountRisk().ObserveAPIKeyCreated(ctx, user, apiKeyID)
   ```

规则本身不碰数据库、不发邮件、不写审计——这些由 `AccountRiskService.Evaluate`
统一完成，因此新增规则不会影响既有规则的处置链路。

## 6. 相关前置配置

- 注册邮箱后缀白名单：`registration_email_suffix_whitelist`（`["@qq.com", ...]`，
  支持 `*.example.com` 通配）。白名单非空时，未命中域名直接拒绝注册。
- 邮箱验证：`email_verify_enabled`。
- 人机验证：`turnstile_enabled`（Cloudflare Turnstile）。
- 内容审计自动封禁：`content_moderation_config`（与本文框架相互独立，按违规次数封禁）。

## 7. 管理端入口

管理端侧边栏 → **账号风控**（`/admin/account-risk`），分两块：

- **规则配置**：总开关 + 四条规则的开关、阈值、处置动作，保存后约 30s 生效。
  后端会对越界值做收敛（例如窗口上限 24h、锁定时长上限 24h），页面回显的是落库后的真实值。
- **命中记录**：直接读操作日志（action 前缀 `security.risk.`），可按「已封禁 / 注册被拦 /
  登录锁定 / 仅留痕」筛选。记录里能看到规则名、阈值、UA、路径，用于复盘误判。

对应接口（均为管理员权限）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/account-risk/config` | 读配置 + 已注册规则名 |
| PUT | `/api/v1/admin/account-risk/config` | 保存配置（归一化后回显） |
| GET | `/api/v1/admin/account-risk/events` | 分页查命中记录，`action` 留空取全部 |

## 8. iframe 嵌入令牌（配合自定义菜单）

自定义菜单 / 首页内容把第三方页面嵌入工作台时，**不要再把会话 JWT 拼进 iframe URL**。
改用受限令牌：

1. 前端调 `POST /api/v1/user/embed-token`，body `{"target_url": "https://第三方/页面"}`；
2. 后端校验该 URL 的 origin 已在 iframe 白名单内（与 CSP `frame-src` 同一份来源），
   签发 5 分钟有效、`scope=embed` 的窄权限令牌并返回；
3. 前端把返回的 `token` 拼进 iframe URL（形式与原来一致，第三方无需改动）。

约束：

- 有效期 5 分钟，泄露窗口从 24 小时缩到 5 分钟；
- 只能访问 `GET /api/v1/user/profile`，其他任何路由一律 403 `EMBED_SCOPE_FORBIDDEN`；
- 不携带会话 ID 与 IP/UA 指纹（第三方服务端来源 IP 必然不同，不能绑）；
- 签发时校验目标 origin，避免本接口变成"给任意站点签发凭证"。

> 该能力当前只在后端就绪，前端自定义菜单仍在使用会话 JWT；切换到 embed-token
> 需要改 `frontend/src/utils/embedded-url.ts` + `CustomPageView.vue`（尚未执行）。
