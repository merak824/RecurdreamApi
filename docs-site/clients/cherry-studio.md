# Cherry Studio 接入

Cherry Studio 支持自定义模型服务，推荐按 OpenAI 兼容方式接入 Recurdream。

## 推荐配置

| 配置项 | 填写 |
| --- | --- |
| Provider | OpenAI Compatible / 自定义 OpenAI |
| API Key | Recurdream API Key |
| API Host | `https://recurdream.com/v1` |
| Model | 后台可用模型名 |

## 操作步骤

1. 打开 Cherry Studio 设置。
2. 进入 **模型服务**。
3. 新增一个自定义 OpenAI 兼容服务。
4. 填写 API Host 和 API Key。
5. 添加模型名。
6. 在聊天页面选择该模型测试。

## 建议

- 每个 Cherry Studio 设备可以使用单独 API Key。
- 若你给 Key 设置了额度，额度耗尽后会停止响应。
- 如果模型名显示正常但请求失败，优先检查余额和分组权限。
