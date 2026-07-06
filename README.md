# 递归梦境API

递归梦境API 是一个面向个人、团队和中转站运营场景的 AI API 网关系统。项目基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 二次开发，当前主线已升级到 Sub2API `v0.1.144`，并保留 Recurdream 的本地定制。

## 当前版本

- 当前版本：`v0.1.144`
- 本仓库主分支：`main`
- 上游项目：[Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- 上游最新同步版本：[v0.1.144](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.144)
- Recurdream 定制发布 Tag：`v0.1.144`

## Release List

这里记录递归梦境API同步上游和本地定制的主要发布节点，方便在 GitHub 主页直接查看版本变化。

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

### 2026-06-07 - 递归梦境API图片工作台更新

- 新增用户侧「图片工作台」页面。
- 新增访问路径：`/image-studio`。
- 左侧菜单新增「图片工作台」入口。
- 支持选择当前用户已有的 API Key。
- 优先展示 OpenAI 分组且已开启生图权限的 Key。
- 支持文生图，调用 `/v1/images/generations`。
- 支持上传参考图改图，调用 `/v1/images/edits`。
- 支持模型、尺寸、张数、质量、背景、输出格式、风格参数。
- 支持图片结果预览、下载、复制和打开。

## Tags

本仓库维护以下发布 Tag，用于对应主分支的可部署状态：

| Tag | 说明 |
| --- | --- |
| `v0.1.144` | 当前最新版本，Sub2API `v0.1.144` + Recurdream 定制 |
| `v0.1.143` | 同步 Sub2API `v0.1.143` 的过渡版本 |
| `v0.1.142` | 同步 Sub2API `v0.1.142` 的过渡版本 |
| `v0.1.137` | 升级前的 Recurdream 基线版本 |

## Recurdream 定制

- 「使用手册」：侧边栏外部导航，打开飞书文档。
- 「图片工作台」：侧边栏外部导航，暂跳转到 `https://image.recurdream.com`。
- 备份脚本：保留服务器每日/每周备份与本地拉取备份脚本。
- 热部署：保留 Docker Compose blue/green 热部署流程。

## 项目定位

这是一个可自部署的 AI API 中转站后台，用于统一管理上游账号、用户 API Key、分组权限、用量计费和请求转发。

适合这些场景：

- 搭建自己的 AI API 中转站
- 管理多个 OpenAI、Claude、Gemini、Antigravity 账号
- 给不同用户或套餐分配不同模型和倍率
- 统计用户用量、账号成本和渠道状态
- 提供 OpenAI-compatible、Claude-compatible、Gemini-compatible 接口
- 为图片模型提供专用生图分组和外部图片工作台入口

## 主要功能

- 多平台账号管理：支持 OpenAI、Anthropic、Gemini、Antigravity 等平台账号。
- API Key 分发：为用户创建、分组、限额、禁用或过期管理 API Key。
- 智能调度：支持账号池调度、失败切换、粘性会话和模型路由。
- 用量计费：记录请求、Token、图片生成、成本和倍率。
- 分组权限：按分组控制模型、倍率、RPM、订阅、图片生成权限。
- 图片生成：支持 `/v1/images/generations` 和 `/v1/images/edits`。
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

OpenAI 兼容接口：

```text
POST /v1/responses
POST /v1/chat/completions
POST /v1/images/generations
POST /v1/images/edits
GET  /v1/models
```

Claude 兼容接口：

```text
POST /v1/messages
POST /v1/messages/count_tokens
```

Gemini 兼容接口：

```text
POST /v1beta/models/{model}:generateContent
GET  /v1beta/models
```

## 使用说明

1. 在后台添加上游账号。
2. 创建分组并绑定账号。
3. 给用户创建 API Key，并绑定对应分组。
4. 用户使用 API Key 调用兼容接口。
5. 如需生图，确保分组开启「允许生图」。
6. 用户可从侧边栏进入「图片工作台」外部站点。
7. 用户可从侧边栏进入「使用手册」飞书文档。

## 说明

本仓库是基于 Sub2API 的自用二开版本，面向递归梦境API网关业务场景维护。请根据自己的实际上游账号、定价策略和合规要求进行部署与配置。
