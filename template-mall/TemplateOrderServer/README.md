# TemplateOrderServer

模板商城核心业务服务：模板元数据、会员、下载资格判断、订单幂等、支付结果消费。仅提供 **gRPC**，不提供业务 HTTP。

## 1. 职责

- 模板创建 / 修改 / 上下架 / 查询
- 用户同步与会员状态
- 下载资格判断（免费 / 会员 / 零售购买 / 未支付复用）
- 订单查询与取消
- HTTP 调用 `PayWebServer` 创建支付单
- 消费 Kafka `template-pay-events`，更新零售订单
- 生成短期下载鉴权地址

## 2. 启动方式

```bash
# 准备 MySQL 库 template_order_db，以及可连通的 Kafka
cp .env.example .env
# 按需修改 .env 后：
set -a && source .env && set +a   # bash
go run ./cmd
```

Windows PowerShell：

```powershell
Get-Content .env | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -notmatch '=') { return }
  $k,$v = $_.Split('=',2); Set-Item -Path "Env:$k" -Value $v.Trim()
}
go run ./cmd
```

默认监听 gRPC：`:50051`（作业「固定端口」表若有官方值，以官方为准并改 `GRPC_ADDR`）。

## 3. 环境变量

见 `.env.example`。关键项：

- `MYSQL_DSN`：连接 `template_order_db`
- `KAFKA_BROKERS` / `KAFKA_TOPIC` / `KAFKA_CONSUMER_GROUP`
- `PAY_BASE_URL`：PayWebServer 根地址
- `STORAGE_PROVIDER`：当前默认 `mock`（本地签名 URL）；后续可接真实 OSS
- `DOWNLOAD_URL_TTL`：下载地址有效期

## 4. 调用关系

```text
TemplateWebServer / TemplateAdminWebServer
        ↓ gRPC
TemplateOrderServer
        ↓ HTTP POST /api/payments
PayWebServer
        ↓ Kafka template-pay-events
TemplateOrderServer（消费者）
```

本服务**不**直接改支付表；支付成功只通过 Kafka 事件驱动订单状态。

## 5. Proto

- 固定入口：`api/proto/template_order.proto`
- 消息拆分：`enums.proto` / `models.proto` / `paging.proto` / `user.proto` / `catalog.proto` / `trade.proto`（保证生成 `.pb.go` 单文件 ≤1000 行）
- 生成代码：`api/gen/templateorder/v1/`

重新生成：

```bash
./scripts/gen-proto.sh
# 或 Windows: scripts/gen-proto.ps1
```

共 **12** 个 RPC，详见 proto 注释。

## 6. 主要设计

- 订单幂等：`orders.idempotency_key` 唯一约束（免费/会员按自然月；零售未支付按用户+模板）
- 支付消费幂等：`kafka_consume_records.event_id` 唯一；业务成功后再 commit offset
- 金额单位：分（整数）
- 取消订单改 `status` + 改写幂等键，释放未支付槽位
- 已取消订单收到支付成功：打错误日志，不改为已支付

## 7. 测试方式

```bash
go test ./internal/service/ -count=1
```

覆盖：免费幂等与并发、未上架拒绝、会员下载、零售复用支付单、取消后再下单、支付成功与重复事件、已取消忽略支付等。

## 8. Docker

```bash
docker build -t template-order-server .
```
