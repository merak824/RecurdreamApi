# Sub2API 热部署指南

这套方案使用 Docker Compose + nginx + blue/green 两个应用槽位：

- `sub2api-blue` 和 `sub2api-green` 轮流承接新版本。
- `sub2api-nginx` 固定占用公网端口，例如 `8080`。
- 新槽位先启动并通过 `/health`，nginx 再切流量。
- 默认切换成功后停止旧槽位，避免后台任务长期双跑。

## 首次使用

```bash
cd deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
bash hot-deploy.sh --build
```

`.env` 里至少要设置：

```bash
POSTGRES_PASSWORD=你的强密码
JWT_SECRET=至少32字节的随机字符串
TOTP_ENCRYPTION_KEY=32字节hex随机字符串
```

可以用下面的命令生成随机值：

```bash
openssl rand -hex 32
```

## 从旧 docker-compose.local.yml 迁移

如果你现在已经用 `docker-compose.local.yml` 跑着 `sub2api`，直接执行：

```bash
cd deploy
bash hot-deploy.sh --build
```

脚本会先启动新的 blue/green 槽位，健康检查通过后，短暂停掉旧的 `sub2api` 容器，让 nginx 接管公网端口。

## 日常更新

从 GitHub 拉新代码并在服务器构建镜像：

```bash
git pull
cd deploy
bash hot-deploy.sh --build
```

低配生产服务器不建议这样做。`--build` 会编译前端和 Go 后端，可能长时间占用 CPU/内存，导致当前线上请求变慢甚至短暂 502。脚本默认会拒绝在物理内存低于 3GB 的机器上执行 `--build`；确实要强行构建时才设置：

```bash
ALLOW_LOW_RESOURCE_BUILD=true bash hot-deploy.sh --build
```

使用已经发布好的镜像：

```bash
cd deploy
SUB2API_IMAGE=ghcr.io/merak824/recurdreamapi:latest bash hot-deploy.sh
```

如果镜像只在本机存在，不需要 pull：

```bash
bash hot-deploy.sh --image sub2api:local --no-pull
```

## 低配服务器推荐流程

如果线上机器只有 1-2GB 内存，推荐在本机或 CI 构建镜像，再上传到服务器加载部署：

```bash
# 在本机或 CI 的仓库根目录构建并导出镜像
IMAGE_TAG=sub2api-hot:0.1.129 bash deploy/build_image.sh --save /tmp/sub2api-hot-0.1.129.tar.gz

# 上传到服务器
scp /tmp/sub2api-hot-0.1.129.tar.gz root@你的服务器:/tmp/

# 在服务器 deploy 目录加载镜像并热部署
cd /data/sub2api
KEEP_OLD=true DRAIN_SECONDS=300 bash hot-deploy.sh --load-image /tmp/sub2api-hot-0.1.129.tar.gz --keep-old
```

这种方式服务器只负责 `docker load`、启动新槽位、健康检查和 nginx 切流，不在生产机上编译代码。

如果当前对话或自动化依赖这个网站自己的 API，建议把部署命令放到后台日志里执行，避免连接中断导致操作悬在终端：

```bash
nohup bash -lc '
cd /data/sub2api
KEEP_OLD=true DRAIN_SECONDS=300 bash hot-deploy.sh --load-image /tmp/sub2api-hot-0.1.129.tar.gz --keep-old
docker compose -f docker-compose.hot.yml --profile blue --profile green ps
' > /tmp/sub2api-hotdeploy.log 2>&1 &

tail -f /tmp/sub2api-hotdeploy.log
```

## 长连接和流式请求

直接运行脚本时，默认切流后会等待 15 秒再停旧槽位；GitHub Actions 日常部署使用 300 秒排空窗口：

```bash
DRAIN_SECONDS=300 bash hot-deploy.sh --build
```

只有需要延长观察或处理超长流式请求时，才让旧槽位继续运行，并在确认后手动停止：

```bash
KEEP_OLD=true bash hot-deploy.sh --build
```

手动停止旧槽位：

```bash
docker compose -f docker-compose.hot.yml --profile blue stop sub2api-blue
docker compose -f docker-compose.hot.yml --profile green stop sub2api-green
```

只停非当前活动槽位即可，当前活动槽位记录在：

```bash
cat .hotdeploy/active-slot
```

## 查看状态

```bash
docker compose -f docker-compose.hot.yml --profile blue --profile green ps
docker compose -f docker-compose.hot.yml logs -f nginx
docker compose -f docker-compose.hot.yml --profile blue logs -f sub2api-blue
docker compose -f docker-compose.hot.yml --profile green logs -f sub2api-green
```

## 注意事项

- 数据库迁移必须兼容旧版本。脚本能保证服务切换不断流，但不能把破坏性 schema 变更自动变成零停机。
- 项目已有 PostgreSQL advisory lock，会避免多实例同时执行迁移。
- 从旧 `docker-compose.local.yml` 迁移到热部署时，默认继续复用 `./data/config.yaml`；不要临时升级 PostgreSQL/Redis 镜像或移动数据目录。
- 如果运行目录里同时存在 `./config.yaml` 和 `./data/config.yaml`，脚本会要求两者一致。旧版本曾经显式挂载 `./config.yaml`，新热部署只挂载 `./data`，两者不一致会导致升级后读错配置。
- GitHub Actions 日常部署默认使用 300 秒排空窗口并自动停止旧槽位。只有长连接或流式请求可能超过 300 秒时，才显式使用 `KEEP_OLD=true`，并在观察完成后手动停止旧槽位。
- 日常热部署只会拉取和替换应用镜像，不会顺手升级 PostgreSQL/Redis。
- 默认不长期保留两个应用实例，主要是为了避免后台任务重复执行。
- 运行时生成的 nginx 配置和活动槽位状态不会提交到 Git。

## 本次线上问题复盘

- 服务器 `/data/sub2api` 是运行目录，不是 Git 仓库；不能直接在该目录 `git pull`。应该在单独源码目录准备代码，或直接部署镜像。
- 线上机器内存较小，现场 `docker build` 编译 Go 大包耗时很长，并给当前服务带来压力。推荐改成本机/CI 构建镜像再上传。
- 旧 compose 曾把 `./config.yaml` 绑定到容器内，新的 blue/green compose 只绑定 `./data`。迁移前必须确认 `./data/config.yaml` 是当前生效配置。
- nginx 出现过一次 `upstream sent too big header`，已在模板中加大 `proxy_buffer_size` 和 `proxy_buffers` 默认值，减少登录/登出响应头过大导致的 502。
- 首轮线上升级如需延长观察，可显式使用 `KEEP_OLD=true`：切到新槽位后旧槽位会保留，确认正常后必须手动停止；常规升级仍使用 300 秒排空后自动停止。
