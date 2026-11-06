# Docker 在线更新代理设计

## 目标

让管理员在任意已登录管理端的电脑上点击“立即更新”，即可把当前 Docker 部署升级到版本检查接口返回的最新正式版本，并在蓝绿热部署完成后看到新版本。现有 systemd/裸机二进制部署继续使用原有的在线替换逻辑。

## 现状与边界

- 当前更新按钮调用 `POST /api/v1/admin/system/update`。
- 二进制更新服务通过下载 GitHub Release 资产并替换当前进程文件实现更新。
- Docker 容器内替换文件不能持久化，也不能重建宿主机上的容器，因此 Docker 环境必须走宿主机部署流程。
- 仓库已有 `deploy/hot-deploy.sh`，负责镜像拉取、蓝绿槽位启动、健康检查、nginx 切流和旧槽位排空。
- 更新任务不接受任意命令、路径或镜像地址；目标版本必须来自当前更新检查结果，并由代理按固定镜像仓库和版本标签构造镜像名。

## 推荐架构

```text
浏览器
  -> 管理端 POST /admin/system/update
  -> 应用服务创建/转发 Docker 更新任务
  -> 宿主机 update-agent Unix socket
  -> 固定执行 deploy/hot-deploy.sh --image <allowlisted-image>:<version>
  -> 蓝绿健康检查与 nginx 切流
  -> 应用服务查询任务状态，前端刷新版本
```

### 宿主机更新代理

- 新增独立的 `recurdream-update-agent` 命令和 systemd 服务。
- 代理通过 Unix socket 提供本机 IPC；socket 由 Docker Compose 以只读路径挂载到应用容器，应用容器不挂载 Docker Socket。
- 代理配置固定部署目录、Compose 文件、热部署脚本、镜像仓库、状态目录和排空时间。
- 代理仅提供创建任务和查询任务两个操作，任务使用单实例锁，重复点击不会并行部署。
- 代理执行失败时保留旧槽位，返回可读错误和日志位置；`hot-deploy.sh` 自身负责流量切换失败回滚。
- 代理服务账号只能执行固定更新程序和读取部署所需配置。安装文档必须明确 Docker 管理权限的风险。

### 应用服务

- 抽象一个 `ContainerUpdateClient`，由 Unix socket HTTP 客户端实现。
- `PerformUpdate` 检测到容器环境时，改为提交容器更新任务，不再返回 `CONTAINER_UPDATE_UNSUPPORTED`。
- 非容器环境继续执行现有二进制下载、校验、替换和重启流程。
- 应用服务不把代理 token 暴露给前端；socket 权限和应用内部路径共同形成访问边界。
- 更新响应返回 `operation_id`、`status` 和目标版本，Docker 模式的 `need_restart` 为 `false`，因为热部署由代理完成。

### 前端

- 点击“立即更新”后显示“正在部署”状态，而不是立即显示二进制更新成功。
- Docker 模式按 `operation_id` 轮询更新状态；轮询期间允许应用短暂重启或 nginx 切流导致的连接失败，并使用有限次数的退避重试。
- 任务成功后清除版本缓存，重新请求版本信息；任务失败显示代理返回的原因和重试入口。
- 仍保留更新日志链接。其他电脑只要能访问管理端并通过管理员认证，即可执行相同流程。

## 接口草案

应用内部客户端与代理使用 Unix socket 上的 HTTP：

```text
POST /v1/update
请求: {"version":"0.2.5"}
响应: {"operation_id":"...","version":"0.2.5","status":"queued"}

GET /v1/update/{operation_id}
响应: {"operation_id":"...","version":"0.2.5","status":"queued|running|succeeded|failed","message":"..."}
```

代理必须校验：

- 版本符合 `major.minor.patch` 格式；
- 版本不低于当前稳定部署版本，除非后续显式增加回滚操作；
- 镜像仓库来自配置的固定 allowlist；
- 同一时间只有一个更新任务；
- 任务超时后状态为 `failed`，不能遗留无限运行的后台进程。

## 配置与安装

新增部署配置示例：

```dotenv
UPDATE_AGENT_SOCKET=/run/recurdream/update-agent.sock
UPDATE_AGENT_IMAGE=weishaw/sub2api
UPDATE_AGENT_DEPLOY_PATH=/data/sub2api
UPDATE_AGENT_DRAIN_SECONDS=300
```

安装流程包括：构建代理、安装 systemd unit、创建 socket 目录和权限、在 `docker-compose.hot.yml` 挂载 socket、执行一次代理健康检查。升级代理本身仍通过服务器运维或 GitHub Actions 完成，避免形成自更新死循环。

## 错误处理与安全

- 代理不可用：更新接口返回“更新代理未安装或未运行”，不再伪装成内部错误。
- 镜像拉取失败：任务失败，流量保持旧槽位。
- 新槽位健康检查失败：停止新槽位，旧槽位继续服务。
- nginx 切流或公共健康检查失败：恢复旧 upstream。
- 应用重启导致轮询连接断开：前端短暂重试并以代理任务状态为准。
- 不允许通过请求参数传入 shell 命令、Compose 文件、部署目录或任意镜像 URL。
- 管理接口仍要求管理员认证和现有系统操作锁；代理 socket 不绑定公网端口。

## 测试与验收

- 代理单元测试：版本校验、固定命令参数、重复任务锁、状态转换、超时和失败记录。
- 应用服务测试：容器模式提交任务，非容器模式仍走原二进制更新；代理不可用时返回结构化错误。
- 前端测试：排队、运行、成功、失败、短暂断线重试和最终版本刷新。
- Shell 测试：安装文件、Compose socket 挂载、systemd unit 和 hot-deploy 参数。
- 隔离验收：使用测试镜像执行一次 blue/green 部署，确认旧槽位失败时公网流量不切换。
- 生产验收：管理员电脑 A 点击更新，电脑 B 刷新页面确认版本一致；数据库、Redis、配置文件和用户数据保持不变。

## 实施顺序

1. 新增代理协议、状态模型和代理命令，先用测试替身验证固定执行边界。
2. 接入应用 `UpdateService` 和管理员接口，保留裸机更新兼容路径。
3. 接入前端轮询和错误展示。
4. 增加 Compose、systemd、安装文档及部署脚本测试。
5. 本地隔离测试、后端构建和前端构建；确认后再由用户安排服务器安装和部署。

