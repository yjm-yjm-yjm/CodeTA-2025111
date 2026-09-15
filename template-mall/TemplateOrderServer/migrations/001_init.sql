-- TemplateOrderServer schema reference (MySQL 8+)
-- DB: template_order_db

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  nickname VARCHAR(128) NOT NULL DEFAULT '',
  is_member TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_users_user_id (user_id),
  KEY idx_users_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  template_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  file_type VARCHAR(16) NOT NULL,
  price_fen BIGINT NOT NULL DEFAULT 0,
  price_type VARCHAR(16) NOT NULL,
  status VARCHAR(16) NOT NULL,
  storage_provider VARCHAR(32) NOT NULL,
  bucket VARCHAR(128) NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  original_filename VARCHAR(255) NOT NULL,
  file_size BIGINT NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_templates_template_id (template_id),
  KEY idx_templates_status (status),
  KEY idx_templates_file_type (file_type),
  KEY idx_templates_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  template_id VARCHAR(64) NOT NULL,
  order_type VARCHAR(16) NOT NULL,
  status VARCHAR(32) NOT NULL,
  price_fen_snapshot BIGINT NOT NULL,
  billing_month VARCHAR(7) NOT NULL DEFAULT '',
  idempotency_key VARCHAR(128) NOT NULL,
  pay_order_id VARCHAR(64) NOT NULL DEFAULT '',
  pay_url VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_orders_order_id (order_id),
  UNIQUE KEY uk_orders_idempotency_key (idempotency_key),
  KEY idx_orders_user (user_id),
  KEY idx_orders_template (template_id),
  KEY idx_orders_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS kafka_consume_records (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(128) NOT NULL,
  order_id VARCHAR(64) NOT NULL,
  pay_order_id VARCHAR(64) NOT NULL DEFAULT '',
  payload TEXT NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY uk_kafka_event_id (event_id),
  KEY idx_kafka_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
