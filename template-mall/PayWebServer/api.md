# PayWebServer HTTP API

Base URL 默认：`http://127.0.0.1:8082`

## 1. 创建支付单

`POST /api/payments`

请求：

```json
{
  "order_id": "ord...",
  "amount_fen": 1200,
  "user_id": "u1",
  "subject": "模板名称"
}
```

响应：

```json
{
  "pay_order_id": "pay...",
  "order_id": "ord...",
  "amount_fen": 1200,
  "pay_url": "http://127.0.0.1:8082/pay/pay...",
  "status": "unpaid"
}
```

相同 `order_id` 重复创建时返回原支付单（含相同 `pay_order_id` / `pay_url`）。

## 2. 支付回调（固定路径）

`POST /api/paycallback`（无需鉴权）

JSON 示例：

```json
{
  "callback_id": "cb-unique-1",
  "pay_order_id": "pay...",
  "order_id": "ord...",
  "amount_fen": 1200
}
```

- 重复 `callback_id`：幂等成功，不重复发 Kafka
- `amount_fen` 与支付单不一致：`400`
- 成功后发送 Kafka，并标记支付单 `paid`

## 3. 模拟支付页

- `GET /pay/:pay_order_id`：HTML 页面
- 页面点「确认支付」时会 **POST `/api/paycallback`**（不另开内部确认接口），与作业要求一致

## 4. 健康检查

`GET /healthz` → `{"ok":true}`
