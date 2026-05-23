# Tavo 接入

Tavo 是 AI 聊天软件，如果它支持自定义 OpenAI 兼容接口，就可以连接 Recurdream。

## 推荐配置

在 Tavo 的模型服务或 API 设置里，选择类似下面的配置：

| 配置项 | 填写 |
| --- | --- |
| 服务商 | OpenAI Compatible / 自定义 OpenAI |
| API Key | Recurdream 后台创建的 API Key |
| Base URL | `https://recurdream.com/v1` |
| 模型 | 按后台可用模型填写，例如 `gpt-4o-mini` |

如果 Tavo 的 Base URL 输入框说明里已经包含 `/v1/chat/completions`，则 Base URL 改成：

```text
https://recurdream.com
```

## 配置步骤

1. 打开 Tavo 设置。
2. 找到模型服务、API 服务商或自定义模型入口。
3. 新增一个 OpenAI 兼容服务。
4. 填写 API Key 和 Base URL。
5. 填写或刷新模型列表。
6. 新建聊天，发送一句测试消息。

## 模型怎么选

优先在 Recurdream 后台查看当前 Key 可用的模型。若 Tavo 支持手动填写模型名，可以直接输入模型名。

常见示例：

```text
gpt-4o-mini
gpt-4.1
claude-sonnet-4-5
gemini-2.5-pro
```

实际可用模型以你的账号后台展示为准。

## 常见问题

### Tavo 提示接口错误

先检查 Base URL。若填写 `https://recurdream.com/v1` 报 404，改成 `https://recurdream.com`；反过来也一样。

### Tavo 刷不出模型

可以先手动填写模型名测试。部分客户端不会自动调用模型列表，或者模型列表接口与聊天接口配置分开。

### 能打开对话但回复失败

检查余额、Key 所属分组、模型权限，以及是否触发速率限制。
