# TemplateWebServer

C 端 BFF：自行注册/登录并签发 JWT，参数校验后通过 gRPC 调用 `TemplateOrderServer`。不连接业务数据库，不实现订单核心逻辑。

## 1. 职责

- 注册 / 登录 / `GET /api/me`
- 已上架模板列表、下载、我的订单、取消未支付零售订单
- JWT 鉴权；gRPC 调用设置超时

## 2. 启动

```powershell
cd TemplateWebServer
# 需先启动 TemplateOrderServer
go run ./cmd
```

默认 HTTP `:8080`，Order gRPC `127.0.0.1:50051`。

## 3. 环境变量

见 `.env.example`。

- `AUTH_USERS_FILE`：本地账号文件（bcrypt），默认 `./data/users.json`
- `JWT_SECRET` / `JWT_TTL`
- `ORDER_GRPC_ADDR` / `GRPC_TIMEOUT`

## 4. 调用关系

```text
TemplateFrontend → HTTP → TemplateWebServer → gRPC → TemplateOrderServer
```

账号凭证仅存本服务本地文件；业务用户通过 `UpsertUser` 同步到 OrderServer。

## 5. Proto 客户端

为避免跨项目引用内部代码，本目录自备 `api/proto` 与生成代码 `api/gen`（从 OrderServer 契约同步）。变更 proto 后请两边一起更新。

## 6. 测试

```bash
go test ./internal/auth/ -count=1
```

## 7. Docker

```bash
docker build -t template-web-server .
```
