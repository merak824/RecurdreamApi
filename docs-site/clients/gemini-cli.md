# Gemini CLI 接入

Gemini CLI 或 Gemini 兼容客户端可以通过 Recurdream 的 Gemini 兼容入口使用。

## 基础配置

不同 Gemini 客户端配置项名称不同，核心是：

| 配置项 | 填写 |
| --- | --- |
| API Key | Recurdream API Key |
| Base URL | `https://recurdream.com` |
| API Version | `v1beta` |
| Model | 后台可用 Gemini 模型 |

## 常用端点

```text
GET  https://recurdream.com/v1beta/models
POST https://recurdream.com/v1beta/models/{model}:generateContent
POST https://recurdream.com/v1beta/models/{model}:streamGenerateContent?alt=sse
```

## 注意事项

- API Key 必须分配到 Gemini 或支持 Gemini 的分组。
- Gemini 客户端通常对 Base URL 和 API Version 更敏感，遇到 404 时先检查路径。
- 如果模型列表为空，可以手动填写后台确认可用的模型名。
