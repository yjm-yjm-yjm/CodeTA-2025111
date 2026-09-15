# TemplateAdminWebServer

B 端 BFF：公司统一 Auth（WPS OAuth）登录即管理员；生成对象存储上传凭证并确认对象存在；经 gRPC 调用 `TemplateOrderServer`。不中转模板文件内容，不直连业务库。

## 1. 职责

- WPS OAuth / Mock 登录，签发管理员 JWT
- OSS 鉴权直传凭证 + 确认对象存在
- 模板创建/修改/上下架/列表
- 订单列表、设置会员

## 2. 启动

```powershell
cd TemplateAdminWebServer
# 需先启动 TemplateOrderServer
go run ./cmd
```

默认 HTTP `:8081`，管理端前端约定 `http://127.0.0.1:3001`。

## 3. 环境变量

见 `.env.example`。

| 变量 | 说明 |
|---|---|
| `AUTH_MODE` | `mock`（本地）或 `wps`（正式） |
| `WPS_*` | OAuth client/回调等（`AUTH_MODE=wps` 时必填） |
| `OSS_PROVIDER` | `mock` 或 `aliyun` |
| `OSS_*` | 阿里云 Endpoint/AK/SK/Bucket |

正式环境：`AUTH_MODE=wps`，登录成功即管理员。本地可 `AUTH_MODE=mock` + `POST /api/auth/mock-login`。

## 4. 上传链路

```text
AdminFrontend → 凭证 → 直传 OSS(或 mock PUT)
            → confirm（服务端 Head 对象）→ gRPC CreateTemplate
```

Object Key 由服务端生成；前端不可任意指定。

## 5. 测试

```bash
go test ./internal/ossstore/ -count=1
```

## 6. Docker

```bash
docker build -t template-admin-web-server .
```
