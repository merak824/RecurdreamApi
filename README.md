# 递归梦境API

递归梦境API 是一个面向个人、团队和中转站运营场景的 AI API 网关系统。项目基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 二次开发，当前主线已升级到 Sub2API `v0.1.165`，并保留 Recurdream 的本地定制。

## 当前版本

- 当前版本：`v0.1.165`
- 本仓库主分支：`main`
- 上游项目：[Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- 上游最新同步版本：[v0.1.165](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.165)
- Recurdream 定制基线：`v0.1.165`

## Release List

这里记录递归梦境API同步上游和本地定制的主要发布节点，方便在 GitHub 主页直接查看版本变化。

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

## TAGS

本仓库维护以下发布 Tag，用于对应主分支的可部署状态：

| Tag | 说明 |
| --- | --- |
| `v0.1.165` | 当前最新基线，Sub2API `v0.1.165` + Recurdream 定制 |
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
