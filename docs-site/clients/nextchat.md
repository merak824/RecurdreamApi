# NextChat 接入

NextChat 支持自定义 OpenAI 接口。桌面版和自部署版本的配置入口可能略有不同。

## 桌面版配置

| 配置项 | 填写 |
| --- | --- |
| API Key | Recurdream API Key |
| Endpoint / Base URL | `https://recurdream.com/v1` |
| Model | 后台可用模型名 |

## 自部署环境变量

如果你使用自部署 NextChat，可以参考：

```text
OPENAI_API_KEY=YOUR_API_KEY
BASE_URL=https://recurdream.com/v1
```

不同版本变量名可能不同，请以 NextChat 当前版本配置项为准。

## 注意事项

- 自部署时不要把 API Key 写进公开仓库。
- 多用户共用一个 Key 时，用量会混在一起。
- 如果 NextChat 支持自定义模型列表，建议只填你实际要使用的模型。
