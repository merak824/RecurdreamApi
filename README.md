# RecurdreamApi

RecurdreamApi 是一个面向个人、团队和中转站运营场景的 AI API 网关系统。项目基于 Sub2API 二次开发，重点增强了多账号调度、分组计费、OpenAI / Claude / Gemini 兼容接入，以及站内图片生成工作台能力。

## 项目定位

这是一个可自部署的 AI API 中转站后台，用于统一管理上游账号、用户 API Key、分组权限、用量计费和请求转发。

适合这些场景：

- 搭建自己的 AI API 中转站
- 管理多个 OpenAI、Claude、Gemini、Antigravity 账号
- 给不同用户或套餐分配不同模型和倍率
- 统计用户用量、账号成本和渠道状态
- 提供 OpenAI-compatible、Claude-compatible、Gemini-compatible 接口
- 为 GPT 图片模型提供专用生图分组和图片工作台

## 主要功能

- 多平台账号管理：支持 OpenAI、Anthropic、Gemini、Antigravity 等平台账号。
- API Key 分发：为用户创建、分组、限额、禁用或过期管理 API Key。
- 智能调度：支持账号池调度、失败切换、粘性会话和模型路由。
- 用量计费：记录请求、Token、图片生成、成本和倍率。
- 分组权限：按分组控制模型、倍率、RPM、订阅、图片生成权限。
- 图片生成：支持 `/v1/images/generations` 和 `/v1/images/edits`，并提供站内图片工作台。
- 后台管理：提供用户、账号、分组、渠道、订单、使用记录、风控等管理页面。
- 热部署：支持 Docker Compose blue/green 热部署，降低线上更新中断风险。

## 新增：图片工作台

本仓库已加入用户侧图片工作台：

- 路由：`/image-studio`
- 菜单：用户侧边栏「图片工作台」
- 支持选择当前用户已有 API Key
- 自动优先展示 OpenAI 且已开启生图权限的 Key
- 支持文生图：调用 `/v1/images/generations`
- 支持参考图改图：上传参考图后调用 `/v1/images/edits`
- 支持参数：模型、尺寸、张数、质量、背景、输出格式、风格
- 支持结果预览、下载、复制和打开图片

推荐配套配置：

```text
分组平台：OpenAI
允许生图：开启
图片模型：gpt-image-2
接口地址：https://你的域名/v1/images/generations
```

站内图片工作台会自动使用：

```text
https://你的域名/v1
```

并根据是否上传参考图自动选择 `/images/generations` 或 `/images/edits`。

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 后端 | Go, Gin, Ent |
| 前端 | Vue 3, Vite, TypeScript, TailwindCSS |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 部署 | Docker, Docker Compose, Nginx |

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
6. 用户可进入「图片工作台」直接使用自己的 Key 生图。

## 说明

本仓库是基于 Sub2API 的自用二开版本，面向 RecurdreamApi / 极算云 API 中转站业务场景维护。请根据自己的实际上游账号、定价策略和合规要求进行部署与配置。

