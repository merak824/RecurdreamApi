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

从 GitHub 拉新代码并本地构建镜像：

```bash
git pull
cd deploy
bash hot-deploy.sh --build
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

## 长连接和流式请求

默认切流后会等待 15 秒再停旧槽位：

```bash
DRAIN_SECONDS=120 bash hot-deploy.sh --build
```

如果你想让旧槽位继续跑，等长连接自然结束后手动停：

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
- 长连接或流式请求较多时，建议首轮使用 `KEEP_OLD=true` 或把 `DRAIN_SECONDS` 调到 120-300 秒，再手动确认旧槽位可以停。
- 日常热部署只会拉取和替换应用镜像，不会顺手升级 PostgreSQL/Redis。
- 默认不长期保留两个应用实例，主要是为了避免后台任务重复执行。
- 运行时生成的 nginx 配置和活动槽位状态不会提交到 Git。
