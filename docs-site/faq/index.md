# 常见问题

## Base URL 到底填哪个

优先填：

```text
https://recurdream.com/v1
```

如果客户端会自动拼接 `/v1/chat/completions`，则填：

```text
https://recurdream.com
```

## API Key 放哪里

客户端一般有 `API Key` 输入框，直接粘贴 Recurdream 控制台创建的 Key 即可。通常不需要手动加 `Bearer`。

## 401 Unauthorized

常见原因：

- API Key 复制不完整。
- API Key 多了空格或换行。
- API Key 已被删除或禁用。
- 客户端把 Key 填到了错误位置。

## 403 Forbidden

常见原因：

- 当前 Key 没有开通对应模型权限。
- 当前账户没有可用额度。
- 账户余额不足。
- 账号或 Key 被禁用。

## 404 Not Found

通常是 Base URL 路径不对。把 `https://recurdream.com` 和 `https://recurdream.com/v1` 互相切换测试。

## 429 Too Many Requests

表示请求太频繁、并发过高或上游账号限流。可以稍等后重试，或降低客户端并发。

## 模型不存在

检查：

- 模型名是否拼写正确。
- 当前 Key 是否支持该模型。
- 账号是否有该模型的使用权限。
- 客户端是否缓存了旧模型列表。

## 支付成功但余额没变

先刷新页面并查看 **我的订单**。如果仍未到账，请联系站点客服并提供订单号、支付时间和支付金额。

## 可以多个客户端共用一个 Key 吗

可以，但建议分开创建。这样用量更清楚，也方便发现异常后单独停用。

## 文档里的模型名为什么和我实际可用模型不一样

文档里的模型名只是示例。实际可用模型以控制台展示和接口返回为准。
