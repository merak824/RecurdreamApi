# ChatBox 接入

ChatBox 可以通过自定义 OpenAI API Host 接入 Recurdream。

## 推荐配置

| 配置项 | 填写 |
| --- | --- |
| Model Provider | OpenAI API |
| API Key | Recurdream API Key |
| API Host | `https://recurdream.com/v1` |
| Model | 后台可用模型名 |

## 测试

保存配置后，新建会话发送：

```text
你好，请用一句话说明你是谁。
```

如果返回正常，即配置成功。

## 排错

- 报 401：重新复制 API Key，确认没有空格。
- 报 404：将 API Host 在 `https://recurdream.com` 和 `https://recurdream.com/v1` 之间切换测试。
- 报模型不存在：换一个后台确认可用的模型名。
