# TemplateOrderServer 数据库设计

数据库名固定：`template_order_db`。本服务只管理下列表；支付相关表由 `PayWebServer` 管理，禁止跨服务直改对方表。

## 1. users

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | 自增主键 |
| user_id | VARCHAR(64) UK | 业务用户 ID（与 C 端 JWT subject 对齐） |
| nickname | VARCHAR(128) | 昵称 |
| is_member | TINYINT | 当前是否会员 |
| created_at / updated_at / deleted_at | DATETIME | 软删可选 |

## 2. templates

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | 自增主键 |
| template_id | VARCHAR(64) UK | 业务模板 ID |
| name | VARCHAR(255) | 名称 |
| file_type | VARCHAR(16) | ppt/pptx/doc/docx |
| price_fen | BIGINT | 价格（分） |
| price_type | VARCHAR(16) | free / paid |
| status | VARCHAR(16) | draft / on_shelf / off_shelf |
| storage_provider / bucket / object_key | VARCHAR | 对象存储定位 |
| original_filename | VARCHAR(255) | 原始文件名 |
| file_size | BIGINT | 字节数，≤5MB |
| created_at / updated_at / deleted_at | DATETIME | 模板可软删 |

规则：free 时 `price_fen=0`；paid 时 `price_fen>0`。上下架用 `status`，不用删除代替。

## 3. orders

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | 自增主键 |
| order_id | VARCHAR(64) UK | 业务订单号 |
| user_id / template_id | VARCHAR(64) | 关联 |
| order_type | VARCHAR(16) | free / member / retail |
| status | VARCHAR(32) | unpaid / download_ready / cancelled |
| price_fen_snapshot | BIGINT | 下单时价格快照 |
| billing_month | VARCHAR(7) | YYYY-MM，免费/会员月度维度 |
| idempotency_key | VARCHAR(128) UK | 幂等键 |
| pay_order_id / pay_url | VARCHAR | 零售支付信息缓存 |
| created_at / updated_at | DATETIME | 订单不物理删除 |

幂等键约定：

- 免费：`free:{user_id}:{template_id}:{YYYY-MM}`
- 会员：`member:{user_id}:{template_id}:{YYYY-MM}`
- 未支付零售：`retail_unpaid:{user_id}:{template_id}`
- 支付成功后：`retail_paid:{order_id}`
- 取消后：`retail_cancelled:{order_id}`（释放 unpaid 槽）

## 4. kafka_consume_records

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | 自增主键 |
| event_id | VARCHAR(128) UK | 支付事件唯一 ID |
| order_id / pay_order_id | VARCHAR(64) | 定位业务 |
| payload | TEXT | 原始摘要 |
| created_at | DATETIME | 写入时间 |

插入成功才继续更新订单；冲突则视为重复消息，直接成功。

## 5. 索引策略

- `templates(status)`、`templates(file_type)`：列表
- `orders(user_id)`、`orders(template_id)`、`orders(status)`：我的订单 / 购买资格
- 唯一：`order_id`、`idempotency_key`、`event_id`

当前允许 `AutoMigrate` 建表（作业允许）；生产应改用 `migrations/` 下 SQL。
