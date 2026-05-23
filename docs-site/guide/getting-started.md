# 快速开始

本页适合第一次使用 Recurdream 的用户。你只需要完成三件事：注册登录、充值余额、创建 API Key。

## 1. 注册并登录

打开 `https://recurdream.com`，进入登录或注册页面。

如果站点开启了邀请码、邮箱验证或第三方登录，请按页面提示完成。注册后进入控制台，可以看到余额、API Key、用量记录等菜单。

## 2. 充值余额

进入控制台后，打开 **充值/订阅** 页面，选择金额并完成支付。

充值完成后，余额通常会自动到账。若支付成功但余额没有变化，请先刷新页面，再到 **我的订单** 查看订单状态。

## 3. 创建 API Key

进入 **API 密钥** 页面，点击 **创建密钥**。

建议给每个使用场景单独创建一个 Key，例如：

| 场景 | 建议名称 |
| --- | --- |
| Tavo 聊天 | `tavo-main` |
| Cherry Studio | `cherry-studio` |
| Claude Code | `claude-code-work` |
| 测试脚本 | `local-test` |

创建后请立刻复制保存。API Key 通常只完整展示一次。

## 4. 选择客户端协议

不同客户端支持的协议不同，填写方式也不同：

| 客户端类型 | 推荐协议 | Base URL |
| --- | --- | --- |
| Tavo、Cherry Studio、ChatBox、NextChat | OpenAI 兼容 | `https://recurdream.com` |
| Claude Code、Claude 兼容工具 | Anthropic/Claude 兼容 | `https://recurdream.com` |
| Gemini CLI、Gemini 兼容工具 | Gemini 兼容 | `https://recurdream.com` |

::: warning
如果某个客户端要求 Base URL 必须包含 `/v1`，可以填写 `https://recurdream.com/v1`。如果客户端会自动拼接 `/v1/chat/completions`，则填写 `https://recurdream.com` 即可。
:::

## 5. 测试一次请求

OpenAI 兼容客户端可用下面的方式测试：

```bash
curl https://recurdream.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      { "role": "user", "content": "你好，简单介绍一下 Recurdream。" }
    ]
  }'
```

如果返回内容正常，说明 API Key、余额和客户端协议都已经配置成功。

## 下一步

- 查看 [创建 API Key](/guide/api-key)
- 查看 [API 端点与模型](/guide/endpoints)
- 查看 [客户端接入](/clients/)
