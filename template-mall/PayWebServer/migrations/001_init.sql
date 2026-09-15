CREATE TABLE IF NOT EXISTS pay_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  pay_order_id VARCHAR(64) NOT NULL,
  order_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  amount_fen BIGINT NOT NULL,
  subject VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  pay_url VARCHAR(1024) NOT NULL DEFAULT '',
  paid_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_pay_orders_pay_order_id (pay_order_id),
  UNIQUE KEY uk_pay_orders_order_id (order_id),
  KEY idx_pay_orders_user (user_id),
  KEY idx_pay_orders_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS pay_callbacks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  callback_id VARCHAR(128) NOT NULL,
  pay_order_id VARCHAR(64) NOT NULL,
  order_id VARCHAR(64) NOT NULL,
  amount_fen BIGINT NOT NULL,
  raw_body TEXT NOT NULL,
  process_result VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_pay_callbacks_callback_id (callback_id),
  KEY idx_pay_callbacks_pay_order (pay_order_id),
  KEY idx_pay_callbacks_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS pay_event_outbox (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(128) NOT NULL,
  order_id VARCHAR(64) NOT NULL,
  pay_order_id VARCHAR(64) NOT NULL,
  payload TEXT NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_pay_event_outbox_event_id (event_id),
  KEY idx_pay_event_outbox_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
