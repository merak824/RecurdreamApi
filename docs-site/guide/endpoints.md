# API 端点与模型

Recurdream 支持多种客户端协议。实际可用模型以控制台展示和接口返回为准。

## 基础地址

推荐基础地址：

```text
https://recurdream.com
```

如果客户端要求填写完整 OpenAI 兼容地址，可以使用：

```text
https://recurdream.com/v1
```

## OpenAI 兼容端点

常见客户端如 Tavo、Cherry Studio、ChatBox、NextChat 一般使用 OpenAI 兼容协议。

| 用途 | 端点 |
| --- | --- |
| 聊天补全 | `/v1/chat/completions` |
| Responses API | `/v1/responses` |
| 模型列表 | `/v1/models` |

示例：

```bash
curl https://recurdream.com/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Claude/Anthropic 兼容端点

Claude Code 或 Claude 兼容工具通常使用：

| 用途 | 端点 |
| --- | --- |
| Messages | `/v1/messages` |
| 模型列表 | `/v1/models` |
| 用量查询 | `/v1/usage` |

## Gemini 兼容端点

Gemini CLI 或 Gemini 兼容工具通常使用：

| 用途 | 端点 |
| --- | --- |
| 模型列表 | `/v1beta/models` |
| 生成内容 | `/v1beta/models/{model}:generateContent` |
| 流式生成 | `/v1beta/models/{model}:streamGenerateContent` |

## 模型名怎么填

模型名以你的 Recurdream 控制台可用模型为准。一般可以在：

- **API 密钥** 页面查看可用端点或使用说明
- 客户端的模型列表里刷新
- 通过 `/v1/models` 或 `/v1beta/models` 查询

::: tip
如果客户端报“模型不存在”，不一定是模型真的不存在，也可能是 API Key 所属分组没有该模型权限。
:::

## Base URL 填写规则

不同客户端处理路径的方式不一样：

| 客户端要求 | 填写 |
| --- | --- |
| 要求 `Base URL`，自动拼路径 | `https://recurdream.com` |
| 要求 OpenAI `API Base` | `https://recurdream.com/v1` |
| 要求完整请求地址 | 按文档填写完整端点 |

如果不确定，先使用 `https://recurdream.com`。如果报 404，再尝试 `https://recurdream.com/v1`。
