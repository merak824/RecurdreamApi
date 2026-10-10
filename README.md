# 递归梦境API

递归梦境API 是一个面向个人、团队和中转站运营场景的 AI API 网关系统。项目基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 二次开发，当前主线已升级到 Sub2API `v0.2.15`，并保留 Recurdream 的本地定制。

## 当前版本

- 当前版本：`v0.2.15`
- 本仓库主分支：`main`
- 上游项目：[Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- 上游最新同步版本：[v0.2.15](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.15)
- Recurdream 定制基线：`v0.2.15` + 本地业务定制

## 发布记录

这里记录递归梦境API同步上游和本地定制的主要发布节点，方便在 GitHub 主页直接查看版本变化。

### v0.2.15

- 同步上游 Sub2API `v0.2.15`，保留 Recurdream 的红包活动、邀请返利与提现、余额明细、分组利润控制、上游倍率探测、OpenAI TTFT 优化、中文品牌页面、备份脚本及 Docker Compose blue/green 热部署能力。
- 新增 Cline 和 Command Code 平台支持；平台列表、账号表单、模型目录、默认端点、协议分流和上游探测统一改为基于平台清单与 provider profile，减少多协议平台的重复判断。
- 新增 TypeSafe/SystemOne 兼容网关和账号能力，并补充协议路由基准、模型列表、工具调用、开发者消息、思考签名、拒答转换和 WebSocket 图片用量等兼容测试。
- 新增充值优惠阶梯：支持赠金模式和折扣模式，可配置优惠比例、支付币种倍率、到账金额和折后实付金额；新增支付订单赠金字段及数据库迁移 `241`，余额返利按实际充值金额计算，不把赠送额度计入返利基数。
- 用量页面新增输出 TPS 展示，保留本地 OpenAI 首 Token/上游首字及调度诊断；修复零倍率、分数 Token 速率和用量展示边界。
- 优化 Claude 计费、钱包冷却和模型目录缓存；按平台能力展示模型同步入口，并完善 OpenCode Zen 路由、不支持模型处理和用量刷新重试。
- 完善 OpenAI Responses 的上游协议分流、Anthropic 转换、工具选择、developer 消息、旧版 function call、thinking signature 和 web_search 历史回放；修复 web_search 在 Responses Lite 和 OAuth 历史重放中的能力声明。
- 增强支付通知安全、公开支付接口限流、邮箱缓存原子性、备份轮询、弹窗 Escape、插件会话、验证码/合规重置和多处前端异步清理，降低并发竞态和重复提交风险。
- 新增平台清单应用层校验，数据库迁移 `242` 移除旧平台约束，并升级 Go/网络依赖和安全扫描工具以适配 Go `1.27.2`。

### v0.2.11

- 同步上游 Sub2API `v0.2.11`，保留 Recurdream 的红包活动、邀请返利与提现、余额明细、分组利润控制、上游倍率探测、OpenAI TTFT 优化、中文品牌页面、备份脚本及 Docker Compose blue/green 热部署能力。
- 上游新增 OpenCode Go 用量查询、Claude 账号重置额度、幂等请求协调、Claude Code CLI 版本同步，以及相关管理端账号用量和额度操作。
- 上游完善模型广场、账号调度、OpenAI Responses/Claude/Gemini/Codex 兼容、备份归档、渠道监控、批量操作和数据库迁移，并补充对应前后端测试。
- 修复流式响应、工具调用、图片/视频/搜索计费、账号配额恢复、备份恢复和管理端页面交互等问题；本地红包、返利、首字优化和部署定制继续保留。

### v0.1.177

- 同步上游 Sub2API `v0.1.177`，将本地版本标记统一更新为 `0.1.177`，并继续保留 Recurdream 的红包活动、邀请返利与提现、余额明细、分组利润控制、上游倍率探测、OpenAI TTFT 优化、中文品牌页面、备份脚本及 Docker Compose blue/green 热部署能力。
- 修复 OpenAI Responses 上游只返回空 `response.completed` 时被错误记录为成功 `0/0` 用量的问题；该场景现在会进入故障转移，并保留上游请求诊断信息。
- 完善 OpenAI 首字和用量完整性：区分可见输出与终止事件，改进首字超时保护、流式输出暂存、上游首字耗时和语义输出诊断，避免仅有 usage 的终止事件污染 TTFT 统计。
- 完善 Codex 兼容：统一指纹与客户端元数据，增加 turn state 透传与跨账号切换处理，增强原生 Responses compaction v2、上下文压缩和工具调用兼容。
- 新增并完善 `x_search` 能力，补齐 Responses、Chat Completions 兼容桥接、搜索调用计费和相关模型能力识别。
- 扩展 Grok/xAI 支持：补充 Grok 4.6 及订阅档位识别、OAuth/API Key 账号判断、媒体资格和配额处理，并修复相关模型、音频、图像和视频路径的计费与调度边界。
- 完善分组与计费：支持按分组逐模型配置价格，新增分组用量汇总和每日 rollup，补充时区迁移、模型价格解析、图片/搜索等非文本计费回归测试。
- 完善备份与数据管理：新增备份归档处理和 S3 存储测试，增强备份列表、下载、归档和恢复相关的管理端流程；数据库新增分组模型价格及用量汇总迁移 `221` 至 `223`。
- 完善 API Key、模型广场和后台页面：增加 API Key 参数校验，更新分组模型价格编辑、账号用量刷新、用量统计、备份管理、渠道价格和模型广场相关页面及中英文文案。

### v0.1.173 - 2026-08-09

- 同步上游 Sub2API `v0.1.173`，完整保留 Recurdream 的红包活动、邀请返利与提现、余额明细、分组利润控制、上游倍率探测、OpenAI TTFT 优化、中文品牌页面、备份脚本及 Docker Compose blue/green 热部署能力。
- 完善 Grok/xAI 授权：支持 SSO 登录、refresh token 重新授权及 OAuth 会话跨实例共享；隐藏并禁用邮箱密码登录，避免继续使用上游已经废弃的认证方式。
- 扩展 Grok 网关能力：新增图片/视频媒体路由、Voice TTS/STT/Realtime、Custom Voices 管理和 `/v1/web_search`；管理端账号测试支持真实媒体上传与结果预览。
- 完善 Grok 模型和调度：支持可配置的默认文本模型与跨客户端映射、team+model 维度冷却、流式空闲换号、7 天/30 天阈值窗口、命中原因展示，以及 free 档近 24 小时本地用量软门禁。
- 完善 Grok 计费：视频支持按模型族、分辨率和时长定价，搜索按每千次调用计价；修复 pending/失败视频误扣费或重复扣费、流式搜索重复计数、模型级额度牵连其它模型、账单耗尽状态不可恢复等问题。
- 新增被动式渠道监控 V2：基于真实网关流量聚合，不额外向上游发起探活请求；用户端提供健康指标、脉冲矩阵、趋势及模型/错误排行，系统设置可在主动探测 V1 与被动聚合 V2 间切换，并可对普通用户隐藏 RPM/TPM。
- 新增邮箱域名限量注册开关，默认关闭。开启且白名单非空时，非白名单地址按主域名（eTLD+1）归一化，每个主域名限注册一个账户；管理面板补充中英文规则提示。
- 新增上游响应模型审计：记录上游实际返回模型和模型不匹配状态，并优化响应模型观察热路径，便于识别映射异常和对账偏差；与 Recurdream 现有 TTFT、客户端断开及请求上下文诊断字段同时保留。
- 优化 OpenAI OAuth 转发：增加 routing hints，停止注入已废弃的 legacy beta 头；完善流内降载错误在输出前的 failover、响应 flush 边界、容量保护、OAuth count_tokens 回退和 WebSocket 预热续接。
- 修复 Gemini 原生生图按上游实际图片数量计费、自定义模型名记零费用、池模式 429 被错误标记为账号级限流，以及非流式生图在客户端断开后已成功生成却未扣费的问题。
- 修复订阅每日额度未在午夜重置、账单金额精度、上游连接缺少明确拨号超时、系统日志落库失败缺少退避、模型广场组合分组模型缺失等问题。

升级注意：数据库将执行渠道监控 V2、上游响应模型审计及 Grok 视频/音频/搜索定价相关迁移；迁移 `220` 会清理非 Grok 分组遗留的视频定价。Grok 跨厂商模型映射默认关闭，如需保持旧的隐式映射行为，必须在系统设置中显式开启。

### v0.1.171 - 2026-08-05

- 同步上游 Sub2API `v0.1.171`，本次升级不新增数据库迁移，现有利润控制、TTFT、红包和返利数据结构保持不变。
- 人机验证新增腾讯天御和阿里云验证码 2.0；管理端“安全与认证”统一为总开关与服务商单选，Cloudflare Turnstile、腾讯天御和阿里云三者互斥。OAuth 登录启动和 Passkey 登录也纳入验证码保护，阿里云支持中国内地与新加坡线路并在保存时校验 AccessKey。
- Codex 出站身份统一管理：HTTP、透传、WebSocket、探针、模型列表和 Alpha Search 等路径使用一致的 User-Agent、originator 与版本来源，降低请求落入上游降载分桶后账号被错误冷却的概率。
- Codex 客户端版本默认每 6 小时自动同步官方最新稳定版，支持在管理面板关闭；旧的自定义 UA 会保留客户端名和系统指纹，版本段自动按当前生效版本重建。新增 `gateway.disable_codex_originator_normalization` 回滚开关，默认关闭。
- 组合分组支持推理强度上限和映射规则，并覆盖 Messages 路线；仅在客户端明确传入 reasoning effort 时应用，不改变协议桥接的默认推理强度。
- OpenAI 账号卡片缓存并展示剩余重置次数和到期时间，自动剔除过期额度；修复刷新失败中断账号恢复以及客户端中断导致额度恢复不完整的问题。
- 管理端退款流程增加余额不足强制确认：不再只扣可用余额却按全额退款；接口会返回 `require_force`，管理员确认后在同一事务内完成余额扣减和状态流转。直接集成退款 API 的调用方需要兼容这一破坏性变更，Stripe 退款同时增加幂等保护。
- 修复计费失败导致整条用量记录丢失、同一订阅并发续期覆盖天数、前端访问令牌并发刷新偶发掉登录、OpenAI Messages 临时不可调度错误不切号、WebSocket 租约终止事件丢失以及请求取消后仍读取调度快照等问题。
- 修复模型广场图片模型展示价与实际结算口径不一致、提示词安全审计遗漏 Responses `output_text`、待建号 OAuth 验证码消费后仍可重复提交等问题；Grok CLI 固定版本升级到 `0.2.114`，Claude OAuth 地址与 Claude Code CLI 保持一致。
- Recurdream 新增统一余额明细：用户订单页可查看充值/管理员调账、邀请返利转余额和红包奖励，支持类型筛选与分页；管理端用户余额历史同步纳入红包奖励，同时不改变“累计充值”口径。
- 保留 Recurdream 的红包活动、邀请返利和提现、分组利润控制、上游倍率探测、OpenAI TTFT 优化、中文品牌首页、备份脚本及 Docker Compose blue/green 热部署诊断。

升级注意：如自定义过 CSP，请放行腾讯与阿里云验证码所需域名；升级后原 Turnstile 配置保持不变，切换验证码服务商会自动关闭其它服务商；余额不足退款必须由管理员显式强制确认。

### v0.1.170 - 2026-08-02

- 同步上游 Sub2API `v0.1.170`，新增两个默认关闭的数据库迁移：分组利润控制和利润控制相关的 API Key 鉴权缓存失效处理；已有部署启动时会自动执行，未开启新开关时保持升级前行为。
- 新增分组级利润控制：可对 OpenAI、Anthropic、Gemini、Grok、Antigravity 分组按“最低利润率 + 安全缓冲”过滤调度候选，成本倍率过高的账号不参与调度；槽位获取后还会二次复核，粘性会话也必须通过最终检查后才绑定。
- 利润控制使用请求级定价快照，同一请求在排队、重试、切换账号或跨越高峰时段时不会改变判定基准。组合分组以及图片、视频、模型列表、用量、`count_tokens` 等非 Token 计费路径不纳入该门控。
- 新增 `profit-preview` 离线预演工具，可根据导出的分组配置和账号数据，按模型查看默认倍率和最差用户倍率下的可用账号数量，便于上线利润控制前评估影响。
- 上游计费倍率探测扩展到 Anthropic、Gemini、Grok、Antigravity 的 API Key 账号。账号级“自动同步倍率”默认关闭，开启后会把上游声明的基准倍率写回账号倍率，并对写回值执行 `0 < 倍率 <= 100` 校验；同步托管期间禁止手动编辑和批量修改该账号倍率。
- 明确区分调度成本和本地结算：非 OpenAI 平台的上游自报倍率不会直接改变调度排序，实际扣费仍按本地最终倍率计算；账号列表会显示倍率同步来源，写回和探测快照也会留下结构化记录。
- 模型广场优化筛选栏换行对齐、模型排序和表格留白；更新 Codex Auto-review 模型计费费率，继续从用户侧导航进入模型广场查看模型与价格。
- 管理端账号列表支持按当前筛选结果全选，批量删除增加并发限制；首页新增精简展示预设，降低被上游误判为滥用分类的风险。
- 内容审核请求支持通过配置的代理服务器转发；提示词安全审计新增“仅审计最新输入”拦截范围。
- 修复 Anthropic 流式中断漏计费、OpenAI WebSocket 外部取消关闭帧丢失、流内限流未触发 429 重试、pool 模式容量错误不切换、Codex 工具命名空间/`instructions` 丢失、压缩上下文过期恢复、Responses 图片桥接、Grok 冷却与计费 ping、订阅周期、支付设置、图片任务 data URL、SMTP 测试连接等问题。
- 保留上一版本已完成的红包活动、邀请返利/提现、OpenAI 首字（TTFT）优化、TTFT 非事务索引迁移、首页渠道状态和 Recurdream 部署定制。

### v0.1.169 - 2026-08-02

- 同步上游 Sub2API `v0.1.169`，应用 URL 路径校验、代理流熔断等安全修复，并合并 Passkey、模型广场、面板限流、计费和网关改进。
- 新增红包活动：支持管理端创建、编辑、发布、取消和导出活动，支持用户资格校验、随机/等额奖励、参与记录、中奖记录、奖励台账，以及用户侧未读活动提醒。
- 完善 OpenAI 首字（TTFT）优化：记录用户首字、上游首字、首个语义事件、客户端断开和账号切换等诊断信息；支持基于历史 P50/P90 数据的账号调度、灰度比例和探索策略，并在用量页面展示上游首字耗时。
- 优化 TTFT 数据库迁移：将字段迁移与 `CREATE INDEX CONCURRENTLY` 非事务索引迁移拆分，避免索引创建阻塞线上迁移，并增加迁移结构回归测试。
- 重构邀请返利流程：统一可用返利额度，支持任意正数额度转入余额，移除旧的代理/冻结期交互；专属用户可单独配置返利比例和提现权限。
- 新增返利提现能力：用户可选择微信或支付宝并上传收款码，最低提现金额为 1 元；管理端支持提现记录、付款凭证、已支付/驳回处理，驳回时自动退回返利额度。
- 收紧返利计账规则：手动充值不再产生邀请返利，邀请绑定、返利入账、转余额和提现回退均保留审计记录，并补充事务一致性和重复处理测试。
- 优化首页渠道状态摘要：统一处理全部正常、部分正常、性能波动、全部异常和待检测状态，避免首页与监控页面显示不一致。
- 保留现有 Recurdream 品牌页面、外部工具导航、备份脚本和 Docker Compose blue/green 热部署定制。

### v0.1.165 - 2026-07-25

- 同步上游 Sub2API `v0.1.165`。
- 新增 ChatGPT Live 实时会话网关及并发、用量记录支持。
- 完整适配 Anthropic `claude-opus-5` 的模型、定价、上下文和限流配置。
- 优化 Ollama Cloud 用量刷新、公告预览和移动端推广页布局。
- 修复邮箱别名注册绕过、Gemini 图像输出、Grok/OpenAI 池模式重试等问题。
- Recurdream 定制：继续保留品牌页面、渠道状态、外部工具导航、提示词审计和部署流程。

### v0.1.160 - 2026-07-17

- 同步上游 Sub2API `v0.1.160`。
- 新增独立的提示词安全审计引擎和管理端审计控制台，默认关闭。
- 修复 Grok 媒体参考图归一化、媒体资格隔离和调度器缓存资格标记问题。
- 修复被动携带 `image_gen` namespace 的请求误触发 Responses capability 检查的问题。
- 为保存 S3 备份配置补充 step-up TOTP 二次验证门控。
- Recurdream 定制：继续保留外部使用手册、图片工作台、返利/提现、备份和 blue/green 热部署流程。

### v0.1.158 - 2026-07-17

- 同步上游 Sub2API `v0.1.158`。
- 新增管理端用户并发和 RPM 限额批量修改，以及分组一键复制。
- 完善 Grok 官方 API、区域端点和自定义上游的配置、转发和故障回退。
- 优化 Codex 图片桥接、WebSocket 生图状态和模型能力发现。
- Recurdream 定制：继续保留「使用手册」飞书外部导航和「图片工作台」外部导航。
- Recurdream 定制：继续保留返利、代理、提现、备份脚本和 Docker Compose blue/green 热部署流程。

### v0.1.156 - 2026-07-15

- 同步上游 Sub2API `v0.1.156`。
- 新增 Codex Agent Identity 认证及原生 Responses 命名空间兼容，增强工具调用与流式协议处理。
- 完善 Grok OAuth 探测、配额统计、媒体路由及 API Key 账号支持。
- 优化调度器缓存重建、请求失败切换、SSE 输出与前端数据列表缓存。
- Recurdream 定制：侧边栏「使用手册」继续作为飞书文档外部导航。
- Recurdream 定制：侧边栏「图片工作台」继续作为外部导航，跳转到 `https://image.recurdream.com`。
- Recurdream 定制：保留代理/返利/提现接口、服务器备份脚本与 Docker Compose blue/green 热部署流程。

### v0.1.151 - 2026-07-10

- 同步上游 Sub2API `v0.1.151`。
- OpenAI Fast/Flex 策略新增用户级规则，用户专属规则优先于全局规则。
- 修复 Codex 上游 originator 与 User-Agent 错配导致请求 404 的问题。
- 修复 GPT-5.6 计费与用量统计、Grok Responses reasoning effort 参数保留、Codex `image_gen` 命名空间剥离等问题。
- 修复 setup-token 账号未纳入后台自动刷新导致令牌到期后返回 401 的问题。
- Recurdream 定制：侧边栏「使用手册」继续作为飞书文档外部导航。
- Recurdream 定制：侧边栏「图片工作台」继续作为外部导航，跳转到 `https://image.recurdream.com`。
- Recurdream 定制：保留服务器备份脚本与本地拉取备份脚本。

### v0.1.144 - 2026-07-06

- 同步上游 Sub2API `v0.1.144`。
- 修复高并发下用量日志静默丢失导致的对账缺口问题。
- 新增 Anthropic Fable 专属 `7d_oi` 窗口的模型级限流支持。
- 新增 Codex 图像工具策略，支持账号级四态控制。
- Recurdream 定制：侧边栏「使用手册」改为飞书文档外部导航。
- Recurdream 定制：「图片工作台」改为外部导航，暂跳转到 `https://image.recurdream.com`。
- Recurdream 定制：保留服务器备份脚本与本地拉取备份脚本。

### v0.1.143 - 2026-07-02

- 同步上游 Sub2API `v0.1.143`。
- 订阅分组新增高峰时段倍率能力。
- OpenAI WebSocket 新增 `http_bridge` ingress 模式及账号级 WS 选择器。
- 支持恢复已撤销的订阅。
- 用量记录新增 IP 地理位置查询与展示。
- 管理端分组列表支持自定义列显示设置。

### v0.1.142 - 2026-07-01

- 同步上游 Sub2API `v0.1.142`。
- 新增 OpenAI Spark 链接型影子账号。
- 适配 Claude Sonnet 5。
- 增强 Grok 媒体模型路由与图像编辑上传转换。
- 修复账号列表分页、订阅撤销、Codex OAuth 推理内容等问题。

### v0.1.137 - 2026-06-16

- 本地服务器原基线版本。
- 新增 OpenAI 账号重置次数查询/触发重置。
- 新增 `cyber_policy` 硬阻断全链路透传。
- 补充 Claude OAuth、国产 LLM 定价、渠道监控抖动等能力。

## 版本标签

本仓库维护以下发布 Tag，用于对应主分支的可部署状态：

| Tag | 说明 |
| --- | --- |
| `v0.1.177` | 当前最新基线，Sub2API `v0.1.177` + Recurdream 定制 |
| `v0.1.173` | 上一版基线，Sub2API `v0.1.173` + Recurdream 定制 |
| `v0.1.171` | 上一版基线，Sub2API `v0.1.171` + Recurdream 定制 |
| `v0.1.170` | 上一版基线，Sub2API `v0.1.170` + Recurdream 定制 |
| `v0.1.169` | 上一版基线，Sub2API `v0.1.169` + Recurdream 定制 |
| `v0.1.165` | 上一版基线，Sub2API `v0.1.165` + Recurdream 定制 |
| `v0.1.160` | 同步 Sub2API `v0.1.160` 的上一发布版本 |
| `v0.1.158` | 同步 Sub2API `v0.1.158` 的上一发布版本 |
| `v0.1.156` | 同步 Sub2API `v0.1.156` 的上一发布版本 |
| `v0.1.151` | 同步 Sub2API `v0.1.151` 的过渡版本 |
| `v0.1.144` | 同步 Sub2API `v0.1.144` 的过渡版本 |
| `v0.1.143` | 同步 Sub2API `v0.1.143` 的过渡版本 |
| `v0.1.142` | 同步 Sub2API `v0.1.142` 的过渡版本 |
| `v0.1.137` | 升级前的 Recurdream 基线版本 |

## Recurdream 定制

- 「使用手册」：侧边栏外部导航，打开飞书文档。
- 「图片工作台」：侧边栏外部导航，跳转到 `https://image.recurdream.com`。
- 代理/返利/提现：保留运营账号和提现处理所需的服务端与前端类型定制。
- 备份脚本：保留服务器每日/每周备份与本地拉取备份脚本。
- 热部署：保留 Docker Compose blue/green 热部署流程。

## 项目定位

这是一个可自部署的 AI API 中转站后台，用于统一管理上游账号、用户 API Key、分组权限、用量计费和请求转发。

适合这些场景：

- 搭建自己的 AI API 中转站
- 管理多个 OpenAI、Claude、Gemini、Antigravity、Grok 账号
- 给不同用户或套餐分配不同模型和倍率
- 统计用户用量、账号成本和渠道状态
- 提供 OpenAI-compatible、Claude-compatible、Gemini-compatible 接口
- 为图片模型提供专用生图分组和外部图片工作台入口

## 主要功能

- 多平台账号管理：支持 OpenAI、Anthropic、Gemini、Antigravity、Grok 等平台账号。
- API Key 分发：为用户创建、分组、限额、禁用或过期管理 API Key。
- 智能调度：支持账号池调度、失败切换、粘性会话和模型路由。
- 用量计费：记录请求、Token、图片生成、视频生成、成本和倍率。
- 分组权限：按分组控制模型、倍率、RPM、订阅、图片生成权限。
- 图片/批量图片能力：保留上游批量图片基础能力，同时侧边栏使用 Recurdream 外部图片工作台入口。
- 后台管理：提供用户、账号、分组、渠道、订单、使用记录、风控等管理页面。
- 热部署：支持 Docker Compose blue/green 热部署，降低线上更新中断风险。

## 快速部署

推荐使用 Docker Compose 部署。

```bash
cd deploy
cp .env.example .env
chmod 600 .env

# 3. 编辑配置（请使用安全密码和密钥）
nano .env

# 必填配置
POSTGRES_PASSWORD=your_secure_password_here
# JWT 密钥（建议设置，重启后保持登录状态）
JWT_SECRET=your_jwt_secret_here
# TOTP 加密密钥（建议设置，保持双因素认证）
TOTP_ENCRYPTION_KEY=your_totp_key_here
# 可选：管理员账号；留空则首次启动时自动生成并写入日志
ADMIN_EMAIL=
ADMIN_PASSWORD=
# 可选：服务端口
SERVER_PORT=8080
```

修改 `.env` 中的数据库密码、JWT 密钥、管理员账号等配置后启动：

```bash
docker compose up -d
```

如果使用 blue/green 热部署：

```bash
cd deploy
bash hot-deploy.sh --build
```

如果管理员邮箱或密码使用了自动生成值，可从首次启动日志中查看：

```bash
docker compose -f docker-compose.local.yml logs sub2api | grep "Generated admin"
```

低内存服务器建议在本地或 CI 构建镜像，再上传到服务器热部署。

## 常用接口

- OpenAI Chat Completions：`/v1/chat/completions`
- OpenAI Responses：`/v1/responses`
- Claude Messages：`/v1/messages`
- Gemini：`/v1beta/models/...`
- 图片生成：`/v1/images/generations`
- 图片编辑：`/v1/images/edits`

## 使用说明

1. 在后台添加上游账号。
2. 创建分组并绑定账号。
3. 给用户创建 API Key，并绑定对应分组。
4. 用户使用 API Key 调用兼容接口。
5. 如需生图，确保分组开启「允许生图」。
6. 如需查看帮助文档，从侧边栏点击「使用手册」跳转飞书文档。
7. 如需进入图片工作台，从侧边栏点击「图片工作台」跳转外部站点。

## 说明

本仓库是基于 Sub2API 的自用二开版本，面向递归梦境API网关业务场景维护。请根据自己的实际上游账号、定价策略和合规要求进行部署与配置。

## 上游部署补充说明
```bash
# Stop all services
docker compose -f docker-compose.local.yml down

# Restart
docker compose -f docker-compose.local.yml restart

# View all logs
docker compose -f docker-compose.local.yml logs -f

# Remove all data (caution!)
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

---

### Method 3: Apple container (macOS)

Apple-silicon Macs running macOS 26 can run the full Sub2API, PostgreSQL, and Redis stack with Apple `container` 1.1.0 or newer:

```bash
git clone https://github.com/Wei-Shaw/sub2api.git
cd sub2api/deploy
./apple-container.sh init
./apple-container.sh up
./apple-container.sh status
```

This is an operator-managed local workflow; Docker Compose remains the recommended production path. See [deploy/APPLE_CONTAINER.md](deploy/APPLE_CONTAINER.md) for lifecycle commands, persistence, upgrades, and runtime limitations.

---

### Method 4: Build from Source

Build and run from source code for development or customization.

#### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+

#### Build Steps

```bash
# 1. Clone the repository
git clone https://github.com/Wei-Shaw/sub2api.git
cd sub2api

# 2. Install pnpm (if not already installed)
npm install -g pnpm

# 3. Build frontend
cd frontend
pnpm install
pnpm run build
# Output will be in ../backend/internal/web/dist/

# 4. Build backend with embedded frontend
cd ../backend
VERSION="$(./scripts/resolve-version.sh)"
go build -tags embed -ldflags="-X main.Version=${VERSION}" -o sub2api ./cmd/server

# 5. Create configuration file
cp ../deploy/config.example.yaml ./config.yaml

# 6. Edit configuration
nano config.yaml
```

> **Note:** The `-tags embed` flag embeds the frontend into the binary. Without this flag, the binary will not serve the frontend UI.

**Key configuration in `config.yaml`:**

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "sub2api"

redis:
  host: "localhost"
  port: 6379
  username: ""
  password: ""

jwt:
  secret: "change-this-to-a-secure-random-string"
  expire_hour: 24

default:
  user_concurrency: 5
  user_balance: 0
  api_key_prefix: "sk-"
  rate_multiplier: 1.0
```

Additional security-related options are available in `config.yaml`:

- `cors.allowed_origins` for CORS allowlist
- `security.url_allowlist` for upstream/pricing/CRS host allowlists
- `security.url_allowlist.enabled` to disable URL validation (use with caution)
- `security.url_allowlist.allow_insecure_http` to allow HTTP URLs when validation is disabled
- `security.url_allowlist.allow_private_hosts` to allow private/local IP addresses
- `security.response_headers.enabled` to enable configurable response header filtering (disabled uses default allowlist)
- `security.csp` to control Content-Security-Policy headers
- `billing.circuit_breaker` to fail closed on billing errors
- `security.trust_forwarded_ip_for_api_key_acl` enables legacy raw forwarded-header takeover (enabled by default for upgrade compatibility); disable it to enforce `server.trusted_proxies`, which should contain only the exact proxy CIDRs that connect directly to Sub2API
- `security.forwarded_client_ip_headers` configures up to 16 third-party CDN client-IP header names; they are checked in order before the built-in headers only while legacy takeover is enabled
- `turnstile.required` to require Turnstile in release mode

Custom client-IP headers can be set in YAML or as a comma-separated environment variable:

```bash
SECURITY_FORWARDED_CLIENT_IP_HEADERS=True-Client-IP,X-CDN-Client-IP
```

Header names are validated, canonicalized, and de-duplicated. The admin security settings can update the list without a restart; new installations persist YAML/environment defaults and existing installations backfill a missing database value. When legacy takeover is disabled, all custom and built-in raw forwarding headers are ignored and Gin uses only `server.trusted_proxies`. While takeover is enabled, firewall the origin to CDN/proxy addresses and make the edge overwrite every trusted client-IP header. See [`deploy/EDGE_SECURITY.md`](deploy/EDGE_SECURITY.md) for the complete migration and trust-boundary rules.

**⚠️ Security Warning: HTTP URL Configuration**

When `security.url_allowlist.enabled=false`, the system performs minimal URL validation and **allows HTTP URLs by default** (dev-friendly mode; Docker Compose deployments use the same default). For production, explicitly tighten this to HTTPS-only:

```yaml
security:
  url_allowlist:
    enabled: false                # Disable allowlist checks
    allow_insecure_http: false    # HTTPS only (recommended for production)
```

**Or via environment variable:**

```bash
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=false
```

**Risks of allowing HTTP:**
- API keys and data transmitted in **plaintext** (vulnerable to interception)
- Susceptible to **man-in-the-middle (MITM) attacks**
- **NOT suitable for production** environments

**When to use HTTP:**
- ✅ Development/testing with local servers (http://localhost)
- ✅ Internal networks with trusted endpoints
- ✅ Testing account connectivity before obtaining HTTPS
- ❌ Production environments (use HTTPS only)

**Example error for HTTP URLs when `allow_insecure_http: false` is set:**
```
Invalid base URL: invalid url scheme: http
```

If you disable URL validation or response header filtering, harden your network layer:
- Enforce an egress allowlist for upstream domains/IPs
- Block private/loopback/link-local ranges
- Enforce TLS-only outbound traffic
- Strip sensitive upstream response headers at the proxy

#### OpenAI Responses WebSocket ingress limits

`gateway.openai_ws` bounds the lifetime and aggregate count of client-facing
Responses WebSocket sessions. These safeguards apply independently from
per-turn user and account concurrency slots, which are released between turns.

```yaml
gateway:
  openai_ws:
    # Total time to receive and decompress the first client message.
    client_first_message_timeout_seconds: 30
    # Close a client socket idle between completed turns; 0 disables this safeguard.
    ingress_inter_turn_idle_timeout_seconds: 300
    # Distributed API-key limit for live client ingress sessions; 0 disables it.
    max_ingress_connections_per_api_key: 64
```

The first-message timeout is a total read deadline. Deployments that accept
large contexts or image-heavy requests over slower links can raise it to
120-300 seconds. It expires before HTTP bridge routing, so bridge mode does not
override this limit.

The connection cap is coordinated through Redis using a 60-second lease that
is refreshed every 20 seconds. A process that cannot confirm a lease for a
full lease lifetime closes its local WebSocket rather than continuing outside
the global cap.

Enable the v2 mode router before selecting an account-level WS mode such as
`http_bridge`:

```yaml
gateway:
  openai_ws:
    mode_router_v2_enabled: true
```

Or set `GATEWAY_OPENAI_WS_MODE_ROUTER_V2_ENABLED=true` in the environment.
Use `http_bridge` for client-WebSocket/upstream-HTTP operation when rolling out
or mitigating upstream WebSocket issues.

#### Force OpenAI upstream HTTP/SSE

When an egress proxy or network repeatedly reconnects OpenAI Responses
WebSockets, set the global fallback in the persisted deployment configuration:

```yaml
gateway:
  openai_ws:
    force_http: true
```

For Compose and Apple container deployments, the equivalent `.env` setting is:

```bash
GATEWAY_OPENAI_WS_FORCE_HTTP=true
```

This selects HTTP/SSE for OpenAI upstream Responses traffic that would
otherwise use WebSocket. It does not change the client-facing protocol or force
HTTP/1.1; configure `gateway.openai_http2.enabled` (or
`GATEWAY_OPENAI_HTTP2_ENABLED=false`) separately when a proxy is incompatible
with HTTP/2. Unlike the account-level `http_bridge` mode, this global fallback
takes effect without enabling `mode_router_v2_enabled`. Keep the setting in the
deployment's persisted `.env` or `config.yaml`, rather than inside a running
container, so it is read again after an image update or container recreation.

#### ⚠️ Important: Creating the Admin Account

The initial admin account is **only created via the setup wizard** (served at `http://<host>:8080` on first run). The `default.admin_email` / `default.admin_password` fields in `config.yaml` are **not used** to create it — they exist in the template for historical reasons.

Because step 5 above pre-creates `config.yaml`, the setup wizard will be **skipped on first run**: the server detects an existing config and boots straight into normal mode with an empty `users` table, so the first login attempt fails with `invalid email or password`.

**Two ways to create the admin account:**

1. **Recommended — let the wizard generate `config.yaml`:** Skip step 5 (do not run the `cp`). Start `./sub2api` directly; the setup wizard at `http://localhost:8080` walks you through database, Redis, and admin account setup, then writes `config.yaml` for you.

2. **If you already created `config.yaml`:** Temporarily move it aside so the wizard can trigger on first run, then restore it afterwards:
   ```bash
   mv config.yaml config.yaml.bak
   ./sub2api        # wizard runs at http://localhost:8080 and writes a fresh config.yaml
   # stop the server (Ctrl+C) once the wizard completes, then restore your config:
   mv config.yaml.bak config.yaml
   ./sub2api        # restart in normal mode and log in with the admin you just created
   ```

```bash
# 6. Run the application
./sub2api
```

#### Development Mode

```bash
# Backend (with hot reload)
cd backend
go run ./cmd/server

# Frontend (with hot reload)
cd frontend
pnpm run dev
```

#### Code Generation

When editing `backend/ent/schema`, regenerate Ent + Wire:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

---

## Simple Mode

Simple Mode is designed for individual developers or internal teams who want quick access without full SaaS features.

- Enable: Set environment variable `RUN_MODE=simple`
- Default groups are seeded on each startup. Set `SIMPLE_MODE_AUTO_CREATE_DEFAULT_GROUPS=false` (or YAML `simple_mode.auto_create_default_groups: false`) to manage groups yourself. The default is `true`; disabling it does not delete existing groups or change runtime auto-binding or admin concurrency setup.
- Difference: Hides SaaS-related features and skips billing process
- Optional key windows: Set `SIMPLE_MODE_KEY_RATE_LIMIT_ENABLED=true` to enforce each API key's configured 5-hour, daily, and 7-day spending windows. The default is `false`; balance and subscription debit remain bypassed when enabled.
- Window enforcement uses the database as its source of truth and records only API-key window usage. It is a post-request soft cap, so concurrent in-flight requests can overshoot by their final costs. Historical simple-mode usage is not backfilled.
- Security note: In production, you must also set `SIMPLE_MODE_CONFIRM=true` to allow startup

---

## Asynchronous Image Tasks

Long-running OpenAI/Grok image generation and editing can be submitted through `/v1/images/generations/async` or `/v1/images/edits/async`, then polled at `/v1/images/tasks/{task_id}` without holding a CDN connection open. See [Asynchronous Image Tasks](docs/ASYNC_IMAGE_TASKS.md) for request and response examples.

---

## Grok / xAI Support

Sub2API supports both Grok subscription accounts through xAI OAuth and standard xAI API-key accounts. Both account types forward OpenAI-compatible Responses traffic to xAI.

### Supported Scope

- Platform name: `grok`
- Account types: OAuth subscription accounts and xAI API-key accounts
- Public Responses targets: `/v1/responses`, `/responses`, and `/backend-api/codex/responses`, forwarded to the Grok subscription proxy for OAuth accounts or `https://api.x.ai/v1/responses` for API-key accounts
- Public Claude-compatible target: `/v1/messages`, converted to xAI Responses and returned as Anthropic Messages output for Claude CLI style clients
- Public Chat Completions targets: `/v1/chat/completions` and `/chat/completions`, forwarded to the account-type-specific xAI upstream
- Codex CLI style Responses WebSocket ingress is accepted on the Responses targets and bridged to xAI HTTP/SSE Responses upstream
- Text models: `grok-4.5`, `grok-4.3`, `grok-build-0.1`, `grok-composer-2.5-fast`, `grok-4.20-0309-reasoning`, `grok-4.20-0309-non-reasoning`, and `grok-4.20-multi-agent-0309`
- Media targets for Grok groups: `/v1/images/generations`, `/images/generations`, `/v1/images/edits`, `/images/edits`, `/v1/videos/generations`, `/videos/generations`, `/v1/videos/edits`, `/videos/edits`, `/v1/videos/extensions`, `/videos/extensions`, `/v1/videos/{request_id}`, and `/videos/{request_id}`. Generation, editing, and extension requests require the group image-generation permission.
- Media models: `grok-imagine`, `grok-imagine-image-quality`, `grok-imagine-image`, `grok-imagine-image-2.0`, `grok-imagine-edit`, `grok-imagine-video`, and `grok-imagine-video-1.5`
- JSON image-edit and video-generation requests accept image references in `image`, `images`, `reference_images`, and `mask` objects. Use `url` for xAI-compatible payloads; the legacy `image_url` field remains accepted and is normalized to `url` before forwarding.
- Out of scope for this provider: TTS, transcription, browser automation, cookies, and Grok web scraping

### OAuth Configuration

The Grok OAuth flow uses PKCE and does not require committing private secrets. The default client details follow the public xAI OAuth flow used by compatible clients, and every value can be overridden by environment variable:

| Variable | Default |
|----------|---------|
| `XAI_OAUTH_CLIENT_ID` | Public xAI OAuth client ID |
| `XAI_OAUTH_SCOPE` | `openid profile email offline_access grok-cli:access api:access` |
| `XAI_OAUTH_REDIRECT_URI` | `http://127.0.0.1:56121/callback` |
| `XAI_OAUTH_AUTHORIZE_URL` | `https://auth.x.ai/oauth2/authorize` |
| `XAI_OAUTH_TOKEN_URL` | `https://auth.x.ai/oauth2/token` |
| `XAI_BASE_URL` | `https://api.x.ai/v1`; runtime-diagnostics override (account `base_url` controls request forwarding) |
| `XAI_GROK_CLI_VERSION` | `0.2.114`; optional override for the client identity sent to `cli-chat-proxy.grok.com`. The pinned value is also the floor: an override below it is dropped |

Administrators can create Grok OAuth or API-key accounts from the dashboard. OAuth authorization and reauthorization are also available through the admin API:

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/admin/grok/oauth/auth-url` | Generate an xAI OAuth authorization URL |
| `POST /api/v1/admin/grok/oauth/exchange-code` | Exchange a callback URL, query string, or code for OAuth credentials |
| `POST /api/v1/admin/grok/oauth/refresh-token` | Validate or refresh a Grok refresh token |
| `POST /api/v1/admin/grok/accounts/:id/refresh` | Refresh an existing Grok account |

OAuth credential storage reuses the existing account JSON fields: `access_token`, `refresh_token`, `token_type`, `expires_at`, `base_url`, optional `email`, optional `subscription_tier`, and `entitlement_status`. OAuth inference defaults to `https://cli-chat-proxy.grok.com/v1`; existing OAuth accounts that stored the old `https://api.x.ai/v1` default are redirected to the subscription proxy at runtime. Explicit custom upstreams remain unchanged.

For API-key accounts, select **Grok → API Key** in the create-account dialog. The official base URL defaults to `https://api.x.ai/v1`; credentials use the existing `base_url` and `api_key` account fields. OAuth accounts continue to use the subscription flow above.

### Grok Build CLI Configuration

1. In the Sub2API admin dashboard, add either a `grok` OAuth account and complete xAI authorization, or add a Grok API-key account.
2. Create a Grok group, attach the account to it, then create a Sub2API API key assigned to that group.
3. In the user API-key page, click **Use Key** and select **Grok CLI**. The modal generates the correct file and base URL for macOS/Linux or Windows. It also provides an OpenCode configuration on the **OpenCode** tab.
4. If configuring manually, save the following as `~/.grok/config.toml` (Windows: `%USERPROFILE%\.grok\config.toml`):

```toml
[models]
default = "grok"
web_search = "grok"

[model."grok"]
model = "grok-4.5"
base_url = "https://your-sub2api.example.com/v1"
name = "Grok 4.5"
api_key = "sk-your-sub2api-key"
api_backend = "responses"
context_window = 1000000
supports_backend_search = true
```

Back up an existing `config.toml` before merging the entry. The file contains a Sub2API API key, so keep it private and restrict its permissions where supported. Verify the effective configuration and make a smoke request:

```bash
grok inspect
grok -p "Reply with sub2api-ok" -m grok
```

The `base_url` above is the public Sub2API URL ending in `/v1`, not `api.x.ai` or the internal xAI OAuth proxy URL.

### Usage And Quota Display

xAI quota is passive. Sub2API does not invent subscription quota values; it records whitelisted xAI rate-limit headers from successful or rate-limited upstream responses when xAI sends them. Before the first usable upstream response, the dashboard shows quota as unknown and still displays local Sub2API usage stats.

`401` responses temporarily remove accounts with invalid credentials from scheduling. `403` responses are treated as access or entitlement failures instead of token-refresh loops. `429` responses use `Retry-After` or a short cooldown to temporarily remove the account from scheduling.

New Grok image and video generation requests use a media-specific eligibility check. API-key accounts remain eligible. OAuth accounts with explicit Free or forbidden billing evidence are excluded from new media generation. Missing or malformed observations are probed before dispatch; a successful but incomplete billing response is treated as `billing_inconclusive` and remains eligible for backwards compatibility, because an unknown billing schema is not proof that the account lacks media entitlement. Operators can quarantine a known-bad account with `extra.grok_media_eligible=false` or force-enable a verified account with `true`. Imports run the billing-first quota probe proactively. Chat requests and video status lookups are not affected by this media-only quarantine. If no eligible account remains, the media endpoint returns HTTP `503` with error type `grok_media_no_eligible_account`.

Administrators can override automatic media eligibility through the account create/update API by setting `extra.grok_media_eligible` to `false` (exclude) or `true` (force eligible). On update, set it to `null` to remove the override and return to automatic probe-based behavior; omitting the field preserves the current override. A weekly allowance period alone is not treated as a paid tier signal. Successful image responses must contain at least one actual image output; empty HTTP `200` responses trigger account failover instead of being counted and returned as successful generations.

---

## TypeSafe / Jev Support

Sub2API supports TypeSafe API-key accounts through Jev's native, non-streaming System One protocol.

- Platform: `typesafe`; account type: API Key
- Default upstream: `https://api.typesafe.ai`
- Public endpoint: `POST /v1/systemone`
- Model: `jev-latest`, also returned by `/v1/models` for TypeSafe groups
- Questions: `noul`, `choice`, and `score`

Requests and successful responses retain the native System One JSON structure. This endpoint is not compatible with Chat Completions, Responses, Anthropic Messages, or streaming clients.

Question validation follows the TypeSafe OpenAPI wire schema (also used by SDK v0.5.7). `instructions` may be omitted or `null` for all question types. Noul `criteria` may be omitted or `null`; its `true`/`false` descriptions and Choice descriptions accept strings, objects, arrays, or `null`. Score `criteria` must be a non-empty array of string, object, or array descriptions; a single level is valid. SDK integer-keyed Score maps are normalized to arrays by the SDK before sending.

```bash
curl https://your-sub2api.example.com/v1/systemone \
  -H 'Authorization: Bearer sk-your-sub2api-key' \
  -H 'Content-Type: application/json' \
  --data '{"model":"jev-latest","state":"Text to evaluate","questions":{"safety":{"type":"noul","instructions":"Evaluate whether the text is unsafe"}}}'
```

The built-in `jev-latest` price is `$0.042` per million input tokens and `$0` for output tokens. Channel pricing can override both values. Credential, billing, permission, rate-limit, overload, server, and network failures (`401`, `402`, `403`, `429`, `529`, `5xx`, transport errors) use the existing account error policy (including custom error codes and temporary-unschedulable rules) and fail over to another account; request errors (`400`, `413`, and `422`) are returned without retrying another account and never change account state. TypeSafe groups (and Composite requests routed to TypeSafe) reject Messages, Chat Completions, Responses, and count_tokens requests with `404`.

---

## Antigravity Support

Sub2API supports [Antigravity](https://antigravity.so/) accounts. After authorization, dedicated endpoints are available for Claude and Gemini models.

### Dedicated Endpoints

| Endpoint | Model |
|----------|-------|
| `/antigravity/v1/messages` | Claude models |
| `/antigravity/v1beta/` | Gemini models |

### Claude Code Configuration

```bash
export ANTHROPIC_BASE_URL="http://localhost:8080/antigravity"
export ANTHROPIC_AUTH_TOKEN="sk-xxx"
```

### Hybrid Scheduling Mode

Antigravity accounts support optional **hybrid scheduling**. When enabled, the general endpoints `/v1/messages` and `/v1beta/` will also route requests to Antigravity accounts.

> **⚠️ Warning**: Anthropic Claude and Antigravity Claude **cannot be mixed within the same conversation context**. Use groups to isolate them properly.

---

## Project Structure

```
sub2api/
├── backend/                  # Go backend service
│   ├── cmd/server/           # Application entry
│   ├── internal/             # Internal modules
│   │   ├── config/           # Configuration
│   │   ├── model/            # Data models
│   │   ├── service/          # Business logic
│   │   ├── handler/          # HTTP handlers
│   │   └── gateway/          # API gateway core
│   └── resources/            # Static resources
│
├── frontend/                 # Vue 3 frontend
│   └── src/
│       ├── api/              # API calls
│       ├── stores/           # State management
│       ├── views/            # Page components
│       └── components/       # Reusable components
│
└── deploy/                   # Deployment files
    ├── docker-compose.yml    # Docker Compose configuration
    ├── .env.example          # Environment variables for Docker Compose
    ├── config.example.yaml   # Full config file for binary deployment
    └── install.sh            # One-click installation script
```

## Star History

<a href="https://star-history.dera.page/#Wei-Shaw/sub2api&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://star-history.dera.page/svg?repos=Wei-Shaw/sub2api&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://star-history.dera.page/svg?repos=Wei-Shaw/sub2api&type=Date" />
   <img alt="Star History Chart" src="https://star-history.dera.page/svg?repos=Wei-Shaw/sub2api&type=Date" />
 </picture>
</a>

---

## License

This project is licensed under the [GNU Lesser General Public License v3.0](LICENSE) (or later).

Copyright (c) 2026 Wesley Liddick

---

<div align="center">

**If you find this project useful, please give it a star!**

</div>
