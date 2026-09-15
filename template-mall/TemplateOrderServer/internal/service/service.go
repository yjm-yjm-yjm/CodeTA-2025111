package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"template-mall/TemplateOrderServer/internal/membership"
	"template-mall/TemplateOrderServer/internal/model"
	"template-mall/TemplateOrderServer/internal/payclient"
	"template-mall/TemplateOrderServer/internal/pkg/idgen"
	"template-mall/TemplateOrderServer/internal/repository"
	"template-mall/TemplateOrderServer/internal/storage"

	"gorm.io/gorm"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrTemplateOffShelf = errors.New("template not on shelf")
	ErrCancelNotAllowed = errors.New("cancel not allowed")
	ErrForbidden        = errors.New("forbidden")
	ErrAmountMismatch   = errors.New("amount mismatch")
	ErrOrderCancelled   = errors.New("order already cancelled")
)

type PayCreator interface {
	CreatePayment(ctx context.Context, req payclient.CreatePaymentRequest) (*payclient.CreatePaymentResponse, error)
}

type Service struct {
	repo           *repository.Repository
	pay            PayCreator
	signer         storage.Signer
	downloadURLTTL time.Duration
}

func New(repo *repository.Repository, pay PayCreator, signer storage.Signer, downloadURLTTL time.Duration) *Service {
	return &Service{repo: repo, pay: pay, signer: signer, downloadURLTTL: downloadURLTTL}
}

func (s *Service) UpsertUser(ctx context.Context, userID, nickname string) (*model.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user_id required", ErrInvalidArgument)
	}
	return s.repo.UpsertUser(ctx, userID, nickname)
}

func (s *Service) GetUser(ctx context.Context, userID string) (*model.User, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Service) SetMembership(ctx context.Context, userID string, isMember bool) (*model.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user_id required", ErrInvalidArgument)
	}
	return s.repo.SetMembership(ctx, userID, isMember)
}

type SubscribeResult struct {
	Order   *model.Order
	Payment *payclient.CreatePaymentResponse
}

// SubscribeMembership 创建/复用未支付会员订阅单，并拉起模拟支付。
func (s *Service) SubscribeMembership(ctx context.Context, userID, plan string) (*SubscribeResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user_id required", ErrInvalidArgument)
	}
	p, err := membership.GetPlan(plan)
	if err != nil {
		return nil, fmt.Errorf("%w: plan must be month|quarter|year", ErrInvalidArgument)
	}
	if _, err := s.repo.UpsertUser(ctx, userID, ""); err != nil {
		return nil, err
	}

	tid := membership.TemplateID(p.ID)
	key := repository.SubUnpaidIdempotencyKey(userID, p.ID)
	if existing, err := s.repo.GetOrderByIdempotencyKey(ctx, key); err == nil {
		pay := &payclient.CreatePaymentResponse{
			PayOrderID: existing.PayOrderID,
			OrderID:    existing.OrderID,
			AmountFen:  existing.PriceFenSnapshot,
			PayURL:     existing.PayURL,
			Status:     "unpaid",
		}
		if existing.PayOrderID == "" || existing.PayURL == "" {
			created, perr := s.pay.CreatePayment(ctx, payclient.CreatePaymentRequest{
				OrderID:   existing.OrderID,
				AmountFen: existing.PriceFenSnapshot,
				UserID:    userID,
				Subject:   p.Name,
			})
			if perr != nil {
				return nil, perr
			}
			existing.PayOrderID = created.PayOrderID
			existing.PayURL = created.PayURL
			_ = s.repo.SaveOrder(ctx, existing)
			pay = created
		}
		return &SubscribeResult{Order: existing, Payment: pay}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	order := &model.Order{
		OrderID:          idgen.NewBusinessID("ord"),
		UserID:           userID,
		TemplateID:       tid,
		OrderType:        model.OrderTypeRetail,
		Status:           model.OrderStatusUnpaid,
		PriceFenSnapshot: p.PriceFen,
		IdempotencyKey:   key,
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		if _, gerr := s.repo.GetOrderByIdempotencyKey(ctx, key); gerr == nil {
			return s.SubscribeMembership(ctx, userID, plan)
		}
		return nil, err
	}
	created, err := s.pay.CreatePayment(ctx, payclient.CreatePaymentRequest{
		OrderID:   order.OrderID,
		AmountFen: order.PriceFenSnapshot,
		UserID:    userID,
		Subject:   p.Name,
	})
	if err != nil {
		return nil, err
	}
	order.PayOrderID = created.PayOrderID
	order.PayURL = created.PayURL
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}
	return &SubscribeResult{Order: order, Payment: created}, nil
}

func (s *Service) CreateTemplate(ctx context.Context, in CreateTemplateInput) (*model.Template, error) {
	if err := validateTemplatePricing(in.PriceType, in.PriceFen); err != nil {
		return nil, err
	}
	if !allowedFileType(in.FileType) {
		return nil, fmt.Errorf("%w: unsupported file_type", ErrInvalidArgument)
	}
	if in.FileSize <= 0 || in.FileSize > 5*1024*1024 {
		return nil, fmt.Errorf("%w: file_size must be 1B..5MB", ErrInvalidArgument)
	}
	if in.ObjectKey == "" || in.Bucket == "" {
		return nil, fmt.Errorf("%w: storage info required", ErrInvalidArgument)
	}
	status := model.TemplateStatusDraft
	if in.Publish {
		status = model.TemplateStatusOnShelf
	}
	t := &model.Template{
		TemplateID:       idgen.NewBusinessID("tpl"),
		Name:             in.Name,
		FileType:         strings.ToLower(in.FileType),
		PriceFen:         in.PriceFen,
		PriceType:        in.PriceType,
		Status:           status,
		StorageProvider:  in.StorageProvider,
		Bucket:           in.Bucket,
		ObjectKey:        in.ObjectKey,
		CoverObjectKey:   in.CoverObjectKey,
		OriginalFilename: in.OriginalFilename,
		FileSize:         in.FileSize,
	}
	if err := s.repo.CreateTemplate(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

type CreateTemplateInput struct {
	Name             string
	FileType         string
	PriceFen         int64
	PriceType        string
	StorageProvider  string
	Bucket           string
	ObjectKey        string
	CoverObjectKey   string
	OriginalFilename string
	FileSize         int64
	Publish          bool
}

func (s *Service) UpdateTemplate(ctx context.Context, templateID string, name *string, priceFen *int64, priceType *string) (*model.Template, error) {
	t, err := s.repo.GetTemplate(ctx, templateID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if name != nil {
		t.Name = *name
	}
	newType := t.PriceType
	newPrice := t.PriceFen
	if priceType != nil {
		newType = *priceType
	}
	if priceFen != nil {
		newPrice = *priceFen
	}
	if priceType != nil || priceFen != nil {
		if err := validateTemplatePricing(newType, newPrice); err != nil {
			return nil, err
		}
		t.PriceType = newType
		t.PriceFen = newPrice
	}
	if err := s.repo.UpdateTemplate(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) SetTemplateShelf(ctx context.Context, templateID string, publish bool) (*model.Template, error) {
	t, err := s.repo.GetTemplate(ctx, templateID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if publish {
		t.Status = model.TemplateStatusOnShelf
	} else {
		t.Status = model.TemplateStatusOffShelf
	}
	if err := s.repo.UpdateTemplate(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) GetTemplate(ctx context.Context, templateID string) (*model.Template, error) {
	t, err := s.repo.GetTemplate(ctx, templateID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Service) ListTemplates(ctx context.Context, onlyOnShelf bool, fileType string, page, pageSize int) ([]model.Template, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.ListTemplates(ctx, onlyOnShelf, fileType, page, pageSize)
}

type DownloadResult struct {
	Granted  *DownloadGranted
	NeedPay  *PaymentRequired
}

type DownloadGranted struct {
	Order      *model.Order
	URL        string
	ExpireAt   time.Time
}

type PaymentRequired struct {
	Order   *model.Order
	Payment *payclient.CreatePaymentResponse
}

func (s *Service) DownloadTemplate(ctx context.Context, userID, templateID string) (*DownloadResult, error) {
	if userID == "" || templateID == "" {
		return nil, fmt.Errorf("%w: user_id and template_id required", ErrInvalidArgument)
	}
	if _, err := s.repo.UpsertUser(ctx, userID, ""); err != nil {
		return nil, err
	}
	tpl, err := s.repo.GetTemplate(ctx, templateID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if tpl.Status != model.TemplateStatusOnShelf {
		return nil, ErrTemplateOffShelf
	}

	// 1) 已零售购买成功 → 永久下载，复用原订单
	if paid, err := s.repo.FindRetailPaidOrder(ctx, userID, templateID); err == nil {
		return s.grant(ctx, paid, tpl)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 2) 免费模板
	if tpl.PriceType == model.PriceTypeFree {
		order, err := s.getOrCreateMonthlyOrder(ctx, userID, tpl, model.OrderTypeFree, repository.FreeIdempotencyKey)
		if err != nil {
			return nil, err
		}
		return s.grant(ctx, order, tpl)
	}

	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 3) 会员下载付费模板
	if user.IsMember {
		order, err := s.getOrCreateMonthlyOrder(ctx, userID, tpl, model.OrderTypeMember, repository.MemberIdempotencyKey)
		if err != nil {
			return nil, err
		}
		return s.grant(ctx, order, tpl)
	}

	// 4) 非会员零售
	return s.retailDownload(ctx, userID, tpl)
}

func (s *Service) getOrCreateMonthlyOrder(
	ctx context.Context,
	userID string,
	tpl *model.Template,
	orderType string,
	keyFn func(userID, templateID, month string) string,
) (*model.Order, error) {
	month := repository.BillingMonth(time.Now())
	key := keyFn(userID, tpl.TemplateID, month)
	if existing, err := s.repo.GetOrderByIdempotencyKey(ctx, key); err == nil {
		_ = s.repo.SaveOrder(ctx, touch(existing))
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	order := &model.Order{
		OrderID:          idgen.NewBusinessID("ord"),
		UserID:           userID,
		TemplateID:       tpl.TemplateID,
		OrderType:        orderType,
		Status:           model.OrderStatusDownloadReady,
		PriceFenSnapshot: tpl.PriceFen,
		BillingMonth:     month,
		IdempotencyKey:   key,
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		if existing, gerr := s.repo.GetOrderByIdempotencyKey(ctx, key); gerr == nil {
			return existing, nil
		}
		return nil, err
	}
	return order, nil
}

func (s *Service) retailDownload(ctx context.Context, userID string, tpl *model.Template) (*DownloadResult, error) {
	key := repository.RetailUnpaidIdempotencyKey(userID, tpl.TemplateID)
	if existing, err := s.repo.GetOrderByIdempotencyKey(ctx, key); err == nil {
		pay := &payclient.CreatePaymentResponse{
			PayOrderID: existing.PayOrderID,
			OrderID:    existing.OrderID,
			AmountFen:  existing.PriceFenSnapshot,
			PayURL:     existing.PayURL,
			Status:     "unpaid",
		}
		if existing.PayOrderID == "" {
			created, perr := s.pay.CreatePayment(ctx, payclient.CreatePaymentRequest{
				OrderID:   existing.OrderID,
				AmountFen: existing.PriceFenSnapshot,
				UserID:    userID,
				Subject:   tpl.Name,
			})
			if perr != nil {
				return nil, perr
			}
			existing.PayOrderID = created.PayOrderID
			existing.PayURL = created.PayURL
			_ = s.repo.SaveOrder(ctx, existing)
			pay = created
		}
		return &DownloadResult{NeedPay: &PaymentRequired{Order: existing, Payment: pay}}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	order := &model.Order{
		OrderID:          idgen.NewBusinessID("ord"),
		UserID:           userID,
		TemplateID:       tpl.TemplateID,
		OrderType:        model.OrderTypeRetail,
		Status:           model.OrderStatusUnpaid,
		PriceFenSnapshot: tpl.PriceFen,
		BillingMonth:     repository.BillingMonth(time.Now()),
		IdempotencyKey:   key,
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		if _, gerr := s.repo.GetOrderByIdempotencyKey(ctx, key); gerr == nil {
			return s.retailDownload(ctx, userID, tpl)
		}
		return nil, err
	}

	created, err := s.pay.CreatePayment(ctx, payclient.CreatePaymentRequest{
		OrderID:   order.OrderID,
		AmountFen: order.PriceFenSnapshot,
		UserID:    userID,
		Subject:   tpl.Name,
	})
	if err != nil {
		return nil, err
	}
	order.PayOrderID = created.PayOrderID
	order.PayURL = created.PayURL
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}
	return &DownloadResult{NeedPay: &PaymentRequired{Order: order, Payment: created}}, nil
}

func (s *Service) grant(ctx context.Context, order *model.Order, tpl *model.Template) (*DownloadResult, error) {
	signed, err := s.signer.PresignGet(ctx, tpl.Bucket, tpl.ObjectKey, s.downloadURLTTL)
	if err != nil {
		return nil, err
	}
	return &DownloadResult{
		Granted: &DownloadGranted{Order: order, URL: signed.URL, ExpireAt: signed.ExpireAt},
	}, nil
}

// PresignCoverURL 列表/详情用的封面临时 URL（TTL 与下载签名一致）。
func (s *Service) PresignCoverURL(ctx context.Context, tpl *model.Template) (string, error) {
	if tpl == nil || tpl.CoverObjectKey == "" {
		return "", nil
	}
	signed, err := s.signer.PresignGet(ctx, tpl.Bucket, tpl.CoverObjectKey, s.downloadURLTTL)
	if err != nil {
		return "", err
	}
	return signed.URL, nil
}

func (s *Service) ListOrders(ctx context.Context, userID, orderType, status string, page, pageSize int) ([]model.Order, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.ListOrders(ctx, userID, orderType, status, page, pageSize)
}

func (s *Service) GetOrder(ctx context.Context, orderID string) (*model.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return o, err
}

func (s *Service) CancelOrder(ctx context.Context, userID, orderID string) (*model.Order, error) {
	var out *model.Order
	err := s.repo.WithTx(ctx, func(tx *repository.Repository) error {
		o, err := tx.LockOrderByID(ctx, orderID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if o.UserID != userID {
			return ErrForbidden
		}
		if o.OrderType != model.OrderTypeRetail || o.Status != model.OrderStatusUnpaid {
			return ErrCancelNotAllowed
		}
		o.Status = model.OrderStatusCancelled
		if membership.IsSubscriptionTemplate(o.TemplateID) {
			o.IdempotencyKey = repository.SubCancelledIdempotencyKey(o.OrderID)
		} else {
			o.IdempotencyKey = repository.RetailCancelledIdempotencyKey(o.OrderID)
		}
		if err := tx.SaveOrder(ctx, o); err != nil {
			return err
		}
		out = o
		return nil
	})
	return out, err
}

// HandlePaySuccess 消费支付成功事件：校验金额、幂等更新零售订单。
func (s *Service) HandlePaySuccess(ctx context.Context, eventID, orderID, payOrderID string, amountFen int64) error {
	if eventID == "" || orderID == "" {
		return fmt.Errorf("%w: event_id/order_id required", ErrInvalidArgument)
	}
	rec := &model.KafkaConsumeRecord{
		EventID:    eventID,
		OrderID:    orderID,
		PayOrderID: payOrderID,
		Payload:    fmt.Sprintf(`{"amount_fen":%d,"pay_order_id":%q}`, amountFen, payOrderID),
		CreatedAt:  time.Now(),
	}
	inserted, err := s.repo.TryInsertConsumeRecord(ctx, rec)
	if err != nil {
		return err
	}
	if !inserted {
		return nil // 重复消息幂等成功
	}

	return s.repo.WithTx(ctx, func(tx *repository.Repository) error {
		o, err := tx.LockOrderByID(ctx, orderID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if o.Status == model.OrderStatusCancelled {
			return ErrOrderCancelled
		}
		if o.Status == model.OrderStatusDownloadReady {
			return nil
		}
		if o.OrderType != model.OrderTypeRetail || o.Status != model.OrderStatusUnpaid {
			return fmt.Errorf("%w: unexpected order state", ErrInvalidArgument)
		}
		if o.PriceFenSnapshot != amountFen {
			return ErrAmountMismatch
		}
		o.Status = model.OrderStatusDownloadReady
		o.PayOrderID = payOrderID
		if membership.IsSubscriptionTemplate(o.TemplateID) {
			o.IdempotencyKey = repository.SubPaidIdempotencyKey(o.OrderID)
			if err := tx.SaveOrder(ctx, o); err != nil {
				return err
			}
			_, err := tx.SetMembership(ctx, o.UserID, true)
			return err
		}
		o.IdempotencyKey = repository.RetailPaidIdempotencyKey(o.OrderID)
		return tx.SaveOrder(ctx, o)
	})
}

func validateTemplatePricing(priceType string, priceFen int64) error {
	switch priceType {
	case model.PriceTypeFree:
		if priceFen != 0 {
			return fmt.Errorf("%w: free template price must be 0", ErrInvalidArgument)
		}
	case model.PriceTypePaid:
		if priceFen <= 0 {
			return fmt.Errorf("%w: paid template price must > 0", ErrInvalidArgument)
		}
	default:
		return fmt.Errorf("%w: invalid price_type", ErrInvalidArgument)
	}
	return nil
}

func allowedFileType(ft string) bool {
	switch strings.ToLower(ft) {
	case "ppt", "pptx", "doc", "docx", "xls", "xlsx", "pdf":
		return true
	default:
		return false
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func touch(o *model.Order) *model.Order {
	o.UpdatedAt = time.Now()
	return o
}
