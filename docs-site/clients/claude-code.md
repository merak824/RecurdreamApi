# Claude Code 接入

Claude Code 或 Claude/Anthropic 兼容工具可以通过 Recurdream 的 Claude 兼容入口使用。

## 基础配置

常见环境变量形式：

```bash
export ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"
export ANTHROPIC_BASE_URL="https://recurdream.com"
```

Windows PowerShell：

```powershell
$env:ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"
$env:ANTHROPIC_BASE_URL="https://recurdream.com"
```

## API 端点

Claude 兼容请求通常走：

```text
https://recurdream.com/v1/messages
```

模型列表：

```text
https://recurdream.com/v1/models
```

## 注意事项

- API Key 需要支持 Claude/Anthropic 兼容请求。
- 如果页面提示 Claude Code 版本不符合要求，请切换到支持的版本。
- 若出现 403，先检查 Key 权限和账户状态。
