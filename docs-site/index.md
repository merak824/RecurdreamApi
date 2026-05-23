# Recurdream 使用手册

欢迎使用 Recurdream。这里是给用户看的接入手册，按下面步骤操作即可完成注册、充值、创建 API Key 和客户端配置。

## 第一次使用

### 1. 注册并登录

打开 Recurdream 主站：

```text
https://recurdream.com
```

注册账号并登录控制台。如果站点开启了邀请码、邮箱验证或第三方登录，请按页面提示完成。

### 2. 充值余额

进入 **充值/订阅** 页面，选择金额并完成支付。

充值成功后，余额会自动到账。若支付成功但余额未更新，请刷新页面并查看 **我的订单**。

### 3. 创建 API Key

进入 **API 密钥** 页面，点击 **创建密钥**。

建议不同客户端使用不同的 Key，例如：

| 使用场景 | 建议名称 |
| --- | --- |
| Tavo 聊天 | `tavo-main` |
| Cherry Studio | `cherry-studio` |
| ChatBox | `chatbox` |
| Claude Code | `claude-code` |

创建后请立即复制保存。API Key 通常只完整显示一次。

## 客户端怎么填

大多数 AI 聊天客户端选择 **OpenAI Compatible** 或 **自定义 OpenAI** 即可。

| 配置项 | 填写 |
| --- | --- |
| API Key | Recurdream 后台创建的 API Key |
| Base URL | `https://recurdream.com/v1` |
| 模型名 | 以后台可用模型为准 |

如果客户端会自动拼接 `/v1/chat/completions`，Base URL 改成：

```text
https://recurdream.com
```

## 常用教程

- [Tavo 接入](/clients/tavo)
- [Cherry Studio 接入](/clients/cherry-studio)
- [ChatBox 接入](/clients/chatbox)
- [NextChat 接入](/clients/nextchat)
- [Claude Code 接入](/clients/claude-code)
- [Gemini CLI 接入](/clients/gemini-cli)

## 常见问题

| 现象 | 处理 |
| --- | --- |
| 401 Unauthorized | 检查 API Key 是否复制完整、是否已禁用 |
| 403 Forbidden | 检查余额、Key 分组、模型权限 |
| 404 Not Found | 在 `https://recurdream.com` 和 `https://recurdream.com/v1` 之间切换测试 |
| 429 Too Many Requests | 请求过快、并发过高或上游限流，稍后重试 |
| 模型不存在 | 检查模型名和 API Key 所属分组权限 |

更多问题见 [常见问题](/faq/)。
