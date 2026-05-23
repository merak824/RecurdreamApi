# 客户端接入

大部分 AI 聊天软件都可以接入 Recurdream。核心配置只有三项：

| 配置项 | 填写 |
| --- | --- |
| API Key | 你在 Recurdream 后台创建的 Key |
| Base URL | `https://recurdream.com` 或 `https://recurdream.com/v1` |
| 模型名 | 按后台可用模型填写 |

## 按客户端选择

<div class="recur-card-grid">
  <a class="recur-card" href="/clients/tavo">
    <strong>Tavo</strong>
    <span>AI 聊天软件，按 OpenAI 兼容服务接入。</span>
  </a>
  <a class="recur-card" href="/clients/cherry-studio">
    <strong>Cherry Studio</strong>
    <span>桌面客户端，适合配置自定义 OpenAI Provider。</span>
  </a>
  <a class="recur-card" href="/clients/chatbox">
    <strong>ChatBox</strong>
    <span>填写自定义 API Host 和 API Key。</span>
  </a>
  <a class="recur-card" href="/clients/nextchat">
    <strong>NextChat</strong>
    <span>适合自部署或桌面版使用自定义接口。</span>
  </a>
</div>

## 通用 OpenAI 兼容配置

如果客户端没有单独教程，只要它支持 OpenAI Compatible、自定义 OpenAI、第三方 OpenAI 接口，通常都可以这样填：

```text
Provider: OpenAI Compatible
API Key: YOUR_API_KEY
Base URL: https://recurdream.com/v1
Model: gpt-4o-mini
```

有些客户端会自动补 `/v1`，这时 Base URL 改成：

```text
https://recurdream.com
```

## 通用排错

| 现象 | 处理 |
| --- | --- |
| 401 Unauthorized | API Key 错误、复制不完整或已被禁用 |
| 403 Forbidden | Key 未分组、分组无权限、账号被禁用 |
| 404 Not Found | Base URL 多填或少填了 `/v1` |
| 429 Too Many Requests | 并发、速率或上游账号限流 |
| 余额不足 | 充值后再试，或检查 Key 额度是否耗尽 |
