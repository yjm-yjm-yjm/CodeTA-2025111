# PayWebServer

模拟支付服务：创建/复用支付单、接收支付回调、向 Kafka 发送支付成功事件。只提供 HTTP，不提供 gRPC，不修改模板订单表。

## 1. 职责

- `POST /api/payments`：按业务订单号创建或复用支付单
- `POST /api/paycallback`：作业固定回调路径，免鉴权，幂等处理
- `GET /pay/:pay_order_id`：模拟支付页
- 支付成功后向 Topic `template-pay-events` 发消息（Key=业务订单号）

## 2. 启动方式

```powershell
cd PayWebServer
# 配置 .env 后加载环境变量
go run ./cmd
```

默认监听 `:8082`。

## 3. 环境变量

见 `.env.example`。需与 OrderServer 共用库 `template_order_db`，但只操作 `pay_*` 表。

## 4. 调用关系

```text
TemplateOrderServer  --HTTP-->  PayWebServer(/api/payments)
模拟支付页/回调      --HTTP-->  PayWebServer(/api/paycallback)
PayWebServer         --Kafka--> template-pay-events
TemplateOrderServer  <--consume--
```

## 5. 主要设计

- `pay_orders.order_id` 唯一：相同业务订单复用支付单
- `pay_callbacks.callback_id` 唯一：回调幂等
- 金额不一致拒绝并返回错误
- Kafka 消息字段与 OrderServer 消费约定一致：`event_id/pay_order_id/order_id/amount_fen/pay_status/paid_at_unix`
- 单文件代码行数 ≤1000

## 6. 测试

```bash
go test ./internal/service/ -count=1
```

## 7. Docker

```bash
docker build -t pay-web-server .
```
