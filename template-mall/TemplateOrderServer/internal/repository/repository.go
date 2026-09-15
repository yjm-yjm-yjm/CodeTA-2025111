package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"template-mall/TemplateOrderServer/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *gorm.DB { return r.db }

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.User{},
		&model.Template{},
		&model.Order{},
		&model.KafkaConsumeRecord{},
	)
}

func (r *Repository) UpsertUser(ctx context.Context, userID, nickname string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = model.User{UserID: userID, Nickname: nickname, IsMember: false}
		if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if err != nil {
		return nil, err
	}
	if nickname != "" && nickname != user.Nickname {
		user.Nickname = nickname
		if err := r.db.WithContext(ctx).Save(&user).Error; err != nil {
			return nil, err
		}
	}
	return &user, nil
}

func (r *Repository) GetUser(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) SetMembership(ctx context.Context, userID string, isMember bool) (*model.User, error) {
	user, err := r.UpsertUser(ctx, userID, "")
	if err != nil {
		return nil, err
	}
	user.IsMember = isMember
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) CreateTemplate(ctx context.Context, t *model.Template) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *Repository) UpdateTemplate(ctx context.Context, t *model.Template) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *Repository) GetTemplate(ctx context.Context, templateID string) (*model.Template, error) {
	var t model.Template
	if err := r.db.WithContext(ctx).Where("template_id = ?", templateID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) ListTemplates(ctx context.Context, onlyOnShelf bool, fileType string, page, pageSize int) ([]model.Template, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Template{})
	if onlyOnShelf {
		q = q.Where("status = ?", model.TemplateStatusOnShelf)
	}
	if fileType != "" {
		q = q.Where("file_type = ?", fileType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Template
	offset := (page - 1) * pageSize
	if err := q.Order("id desc").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) GetOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	var o model.Order
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) GetOrderByIdempotencyKey(ctx context.Context, key string) (*model.Order, error) {
	var o model.Order
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) FindRetailPaidOrder(ctx context.Context, userID, templateID string) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND template_id = ? AND order_type = ? AND status = ?",
			userID, templateID, model.OrderTypeRetail, model.OrderStatusDownloadReady).
		Order("id asc").
		First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) ListOrders(ctx context.Context, userID, orderType, status string, page, pageSize int) ([]model.Order, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Order{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if orderType != "" {
		q = q.Where("order_type = ?", orderType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Order
	offset := (page - 1) * pageSize
	if err := q.Order("id desc").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) CreateOrder(ctx context.Context, o *model.Order) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *Repository) SaveOrder(ctx context.Context, o *model.Order) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *Repository) WithTx(ctx context.Context, fn func(tx *Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

func (r *Repository) LockOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	var o model.Order
	if err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ?", orderID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) TryInsertConsumeRecord(ctx context.Context, rec *model.KafkaConsumeRecord) (inserted bool, err error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(rec)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func FreeIdempotencyKey(userID, templateID, month string) string {
	return fmt.Sprintf("free:%s:%s:%s", userID, templateID, month)
}

func MemberIdempotencyKey(userID, templateID, month string) string {
	return fmt.Sprintf("member:%s:%s:%s", userID, templateID, month)
}

func RetailUnpaidIdempotencyKey(userID, templateID string) string {
	return fmt.Sprintf("retail_unpaid:%s:%s", userID, templateID)
}

func RetailPaidIdempotencyKey(orderID string) string {
	return fmt.Sprintf("retail_paid:%s", orderID)
}

func RetailCancelledIdempotencyKey(orderID string) string {
	return fmt.Sprintf("retail_cancelled:%s", orderID)
}

func SubUnpaidIdempotencyKey(userID, plan string) string {
	return fmt.Sprintf("sub_unpaid:%s:%s", userID, plan)
}

func SubPaidIdempotencyKey(orderID string) string {
	return fmt.Sprintf("sub_paid:%s", orderID)
}

func SubCancelledIdempotencyKey(orderID string) string {
	return fmt.Sprintf("sub_cancelled:%s", orderID)
}

func BillingMonth(t time.Time) string {
	return t.Format("2006-01")
}
