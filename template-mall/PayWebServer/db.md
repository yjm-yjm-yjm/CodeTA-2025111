# PayWebServer 数据库设计

共用数据库：`template_order_db`。本服务只读写下列表，禁止修改 `orders` / `templates` 等订单服务表。

## 1. pay_orders

| 字段 | 说明 |
|---|---|
| id | 主键 |
| pay_order_id | 支付单号，唯一 |
| order_id | 业务订单号，唯一（复用依据） |
| user_id | 用户 |
| amount_fen | 金额（分） |
| subject | 商品描述 |
| status | unpaid / paid |
| pay_url | 模拟支付页地址 |
| paid_at | 支付时间 |
| created_at / updated_at | 时间 |

## 2. pay_callbacks

| 字段 | 说明 |
|---|---|
| id | 主键 |
| callback_id | 回调唯一 ID（幂等） |
| pay_order_id / order_id | 定位 |
| amount_fen | 回调金额 |
| raw_body | 原始报文 |
| process_result | paid / already_paid / … |
| created_at | 时间 |

## 3. pay_event_outbox

记录已准备发送的 Kafka 事件摘要，便于排障（本作业不强制可靠投递补偿）。

| 字段 | 说明 |
|---|---|
| event_id | 唯一 |
| order_id / pay_order_id | 定位 |
| payload | JSON |
| created_at | 时间 |
