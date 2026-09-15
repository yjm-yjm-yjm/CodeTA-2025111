package model

import "time"

const (
	PayStatusUnpaid = "unpaid"
	PayStatusPaid   = "paid"
)

// PayOrder 支付单。相同业务订单号复用同一支付单。
type PayOrder struct {
	ID         uint64    `gorm:"primaryKey"`
	PayOrderID string    `gorm:"size:64;uniqueIndex;not null"`
	OrderID    string    `gorm:"size:64;uniqueIndex;not null"` // 业务订单号，唯一保证复用
	UserID     string    `gorm:"size:64;not null;index"`
	AmountFen  int64     `gorm:"not null"`
	Subject    string    `gorm:"size:255;not null;default:''"`
	Status     string    `gorm:"size:32;not null;index"`
	PayURL     string    `gorm:"size:1024;not null;default:''"`
	PaidAt     *time.Time
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}

func (PayOrder) TableName() string { return "pay_orders" }

// PayCallback 支付回调去重记录。
type PayCallback struct {
	ID             uint64    `gorm:"primaryKey"`
	CallbackID     string    `gorm:"size:128;uniqueIndex;not null"`
	PayOrderID     string    `gorm:"size:64;not null;index"`
	OrderID        string    `gorm:"size:64;not null;index"`
	AmountFen      int64     `gorm:"not null"`
	RawBody        string    `gorm:"type:text;not null"`
	ProcessResult  string    `gorm:"size:64;not null;default:''"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (PayCallback) TableName() string { return "pay_callbacks" }

// PayEventOutbox 可选：记录已发送的 Kafka 事件，便于排查（非强制 Outbox 模式）。
type PayEventOutbox struct {
	ID         uint64    `gorm:"primaryKey"`
	EventID    string    `gorm:"size:128;uniqueIndex;not null"`
	OrderID    string    `gorm:"size:64;not null;index"`
	PayOrderID string    `gorm:"size:64;not null"`
	Payload    string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (PayEventOutbox) TableName() string { return "pay_event_outbox" }
