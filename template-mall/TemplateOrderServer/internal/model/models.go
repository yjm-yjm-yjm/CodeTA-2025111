package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	PriceTypeFree = "free"
	PriceTypePaid = "paid"

	TemplateStatusDraft    = "draft"
	TemplateStatusOnShelf  = "on_shelf"
	TemplateStatusOffShelf = "off_shelf"

	OrderTypeFree   = "free"
	OrderTypeMember = "member"
	OrderTypeRetail = "retail"

	OrderStatusUnpaid         = "unpaid"
	OrderStatusDownloadReady  = "download_ready"
	OrderStatusCancelled      = "cancelled"
)

type User struct {
	ID        uint64         `gorm:"primaryKey"`
	UserID    string         `gorm:"size:64;uniqueIndex;not null"`
	Nickname  string         `gorm:"size:128;not null;default:''"`
	IsMember  bool           `gorm:"not null;default:false"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string { return "users" }

type Template struct {
	ID               uint64         `gorm:"primaryKey"`
	TemplateID       string         `gorm:"size:64;uniqueIndex;not null"`
	Name             string         `gorm:"size:255;not null"`
	FileType         string         `gorm:"size:16;not null;index"`
	PriceFen         int64          `gorm:"not null;default:0"`
	PriceType        string         `gorm:"size:16;not null"`
	Status           string         `gorm:"size:16;not null;index"`
	StorageProvider  string         `gorm:"size:32;not null"`
	Bucket           string         `gorm:"size:128;not null"`
	ObjectKey        string         `gorm:"size:512;not null"`
	CoverObjectKey   string         `gorm:"size:512;not null;default:''"`
	OriginalFilename string         `gorm:"size:255;not null"`
	FileSize         int64          `gorm:"not null"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (Template) TableName() string { return "templates" }

// Order 模板订单。取消/支付成功均用 status 表达，不物理删除。
type Order struct {
	ID               uint64    `gorm:"primaryKey"`
	OrderID          string    `gorm:"size:64;uniqueIndex;not null"`
	UserID           string    `gorm:"size:64;not null;index:idx_orders_user"`
	TemplateID       string    `gorm:"size:64;not null;index:idx_orders_template"`
	OrderType        string    `gorm:"size:16;not null"`
	Status           string    `gorm:"size:32;not null;index"`
	PriceFenSnapshot int64     `gorm:"not null"`
	BillingMonth     string    `gorm:"size:7;not null;default:''"` // YYYY-MM，免费/会员月度幂等
	// IdempotencyKey 唯一约束参与幂等：
	// free:{uid}:{tid}:{YYYY-MM} | member:{uid}:{tid}:{YYYY-MM} | retail_unpaid:{uid}:{tid}
	// 零售支付成功或取消后会改写，释放 unpaid 槽位。
	IdempotencyKey string    `gorm:"size:128;uniqueIndex;not null"`
	PayOrderID     string    `gorm:"size:64;not null;default:''"`
	PayURL         string    `gorm:"size:1024;not null;default:''"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (Order) TableName() string { return "orders" }

// KafkaConsumeRecord 支付事件消费去重。
type KafkaConsumeRecord struct {
	ID        uint64    `gorm:"primaryKey"`
	EventID   string    `gorm:"size:128;uniqueIndex;not null"`
	OrderID   string    `gorm:"size:64;not null;index"`
	PayOrderID string   `gorm:"size:64;not null;default:''"`
	Payload   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (KafkaConsumeRecord) TableName() string { return "kafka_consume_records" }
