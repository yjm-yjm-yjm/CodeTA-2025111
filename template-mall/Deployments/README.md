# Deployments

模板商城本地/演示部署（Docker Compose）。

## 1. 包含组件

| 服务 | 端口 |
|---|---|
| MySQL | 3308（容器内仍为 3306；本机工具连 `127.0.0.1:3308`） |
| Kafka | 9092（容器内 `kafka:29092`） |
| Kafka UI（可选） | 8090 |
| TemplateOrderServer | 50051/gRPC |
| PayWebServer | 8082 |
| TemplateWebServer | 8080 |
| TemplateAdminWebServer | 8081 |

前端不强制进 Compose，本地分别：

- C 端：`TemplateFrontend` → `npm run dev`（3000）
- B 端：`TemplateAdminFrontend` → `npm run dev`（3001）

## 2. 启动前准备

1. 安装并启动 Docker Desktop。
2. 各后端准备本机 `.env`（从 `.env.example` 复制，**不要把密钥写进 example**）：
   - `TemplateOrderServer/.env`
   - `TemplateAdminWebServer/.env`（OSS）
   - `TemplateWebServer/.env`
   - `PayWebServer/.env`
3. OSS 相关只放 `.env`，确保已加入 `.gitignore`。

## 3. 启动

```powershell
cd Deployments
docker compose up -d --build
docker compose ps
```

健康检查：

```powershell
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8081/healthz
curl http://127.0.0.1:8082/healthz
```

## 4. 停止

```powershell
docker compose down
```

保留数据卷：默认不删 volume。清库：

```powershell
docker compose down -v
```

## 5. 联调脚本

仓库提供 `scripts/e2e-smoke.ps1`（管理端 mock 登录 → 上传模板 → C 端注册下载 → 支付回调）。

```powershell
cd Deployments
powershell -File .\scripts\e2e-smoke.ps1
```

本机直跑后端（不进容器）时：各服务会读取工作目录 `.env`；MySQL 请连宿主机映射口 `127.0.0.1:3308`。

## 6. 内置模板

4 个演示模板素材在 `seed/builtin/`。服务起来后执行：

```powershell
powershell -File .\scripts\seed-builtin-templates.ps1
```

## 7. 镜像拉取过慢时

可先用镜像站拉取再打官方 tag，例如：

```powershell
docker pull docker.m.daocloud.io/apache/kafka:3.8.1
docker tag docker.m.daocloud.io/apache/kafka:3.8.1 apache/kafka:3.8.1
```
