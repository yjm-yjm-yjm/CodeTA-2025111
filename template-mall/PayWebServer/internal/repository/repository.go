package repository

import (
	"context"
	"errors"

	"template-mall/PayWebServer/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&model.PayOrder{}, &model.PayCallback{}, &model.PayEventOutbox{})
}

func (r *Repository) WithTx(ctx context.Context, fn func(tx *Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

func (r *Repository) GetByOrderID(ctx context.Context, orderID string) (*model.PayOrder, error) {
	var o model.PayOrder
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) GetByPayOrderID(ctx context.Context, payOrderID string) (*model.PayOrder, error) {
	var o model.PayOrder
	if err := r.db.WithContext(ctx).Where("pay_order_id = ?", payOrderID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) CreatePayOrder(ctx context.Context, o *model.PayOrder) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *Repository) SavePayOrder(ctx context.Context, o *model.PayOrder) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *Repository) LockByPayOrderID(ctx context.Context, payOrderID string) (*model.PayOrder, error) {
	var o model.PayOrder
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("pay_order_id = ?", payOrderID).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) TryInsertCallback(ctx context.Context, c *model.PayCallback) (inserted bool, err error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(c)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *Repository) InsertOutbox(ctx context.Context, e *model.PayEventOutbox) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(e).Error
}

func (r *Repository) UpdateCallback(ctx context.Context, c *model.PayCallback) error {
	return r.db.WithContext(ctx).Model(&model.PayCallback{}).
		Where("callback_id = ?", c.CallbackID).
		Updates(map[string]any{
			"pay_order_id":   c.PayOrderID,
			"order_id":       c.OrderID,
			"amount_fen":     c.AmountFen,
			"process_result": c.ProcessResult,
		}).Error
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
