# TemplateAdminWebServer HTTP API

Base：`http://127.0.0.1:8081`  
鉴权：`Authorization: Bearer <admin_token>`（登录相关与 mock 上传除外）

## 1. 登录

### 1.1 WPS（`AUTH_MODE=wps`）

- `GET /api/auth/login?format=json` → `{ data: { authorize_url } }`（管理端前端直跳 WPS）
- `GET /api/auth/login` → 302 跳转 WPS 授权页
- `GET /api/auth/callback?code=&state=` → 换票拉用户 → 302 到 `ADMIN_FRONTEND_URL?token=...`
- `WPS_REDIRECT_URI` 默认 `http://localhost:3001/api/auth/callback`（须与开放平台登记一致；开发时由管理端 Vite 代理到本服务）

### 1.2 Mock（本地联调，与 `AUTH_MODE=wps` 并存）

`POST /api/auth/mock-login`

```json
{ "admin_id": "mock-admin", "nickname": "MockAdmin" }
```

## 2. 当前管理员

`GET /api/me`

## 3. 上传凭证

`POST /api/upload/credential`

```json
{ "filename": "demo.pptx", "content_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation" }
```

返回 `object_key/upload_url/method/expire_at_unix/...`。  
**签名直传**：`OSS_PROVIDER=aliyun` 时 `upload_url` 为 OSS 预签名 PUT 地址，浏览器直传对象存储（服务端不转发文件体）；启动时会尝试配置桶 CORS。`mock` 时 `upload_url` 指向本服务 `/api/upload/mock`。

## 4. 确认上传并创建模板

`POST /api/upload/confirm`

```json
{
  "object_key": "templates/...",
  "original_filename": "demo.pptx",
  "file_type": "pptx",
  "file_size": 1024,
  "name": "演示模板",
  "price_fen": 0,
  "price_type": "free",
  "publish": true
}
```

服务端先确认对象存在且 ≤5MB，再 gRPC 创建模板。

## 5. 模板管理

- `GET /api/templates`
- `GET /api/templates/:template_id`
- `POST /api/templates`（需已上传并确认）
- `PATCH /api/templates/:template_id`
- `POST /api/templates/:template_id/publish`
- `POST /api/templates/:template_id/unpublish`

## 6. 订单与会员

- `GET /api/orders`
- `POST /api/users/:user_id/membership` body: `{ "is_member": true }`

## 7. 健康检查

`GET /healthz`
