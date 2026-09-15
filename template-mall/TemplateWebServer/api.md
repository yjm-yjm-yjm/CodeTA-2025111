# TemplateWebServer HTTP API

Base：`http://127.0.0.1:8080`  
鉴权：`Authorization: Bearer <token>`（除注册/登录/healthz）

## 1. 注册

`POST /api/auth/register`

```json
{ "phone": "13800138000", "password": "secret123", "nickname": "Alice" }
```

约束：`phone` 须为大陆 11 位手机号（`1[3-9]…`）；`password` 长度 6–64。

响应：`data.token` + `data.user`（含 `phone`）

## 2. 登录

`POST /api/auth/login`

```json
{ "phone": "13800138000", "password": "secret123" }
```

## 3. 当前用户

`GET /api/me`  
返回 `user_id/phone/nickname/is_member`

## 4. 模板列表（仅已上架）

`GET /api/templates?page=1&page_size=20&file_type=pptx`

## 5. 下载

`POST /api/templates/:template_id/download`

成功可下载：

```json
{
  "data": {
    "result": "granted",
    "order_id": "...",
    "download_url": "...",
    "expire_at_unix": 0,
    "order": {}
  }
}
```

需支付：

```json
{
  "data": {
    "result": "payment_required",
    "order": {},
    "payment": { "pay_url": "...", "pay_order_id": "...", "amount_fen": 100 }
  }
}
```

## 6. 我的订单

`GET /api/orders?page=1&page_size=20`

## 7. 取消未支付零售订单

`POST /api/orders/:order_id/cancel`

## 8. 健康检查

`GET /healthz`
