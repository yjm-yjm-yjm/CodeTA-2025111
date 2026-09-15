package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"template-mall/PayWebServer/internal/kafka"
	"template-mall/PayWebServer/internal/model"
	"template-mall/PayWebServer/internal/pkg/idgen"
	"template-mall/PayWebServer/internal/repository"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("not found")
	ErrAmountMismatch  = errors.New("amount mismatch")
)

type Service struct {
	repo          *repository.Repository
	producer      kafka.Producer
	publicBaseURL string
}

func New(repo *repository.Repository, producer kafka.Producer, publicBaseURL string) *Service {
	return &Service{repo: repo, producer: producer, publicBaseURL: publicBaseURL}
}

type CreatePaymentInput struct {
	OrderID   string
	AmountFen int64
	UserID    string
	Subject   string
}

type CreatePaymentResult struct {
	PayOrderID string `json:"pay_order_id"`
	OrderID    string `json:"order_id"`
	AmountFen  int64  `json:"amount_fen"`
	PayURL     string `json:"pay_url"`
	Status     string `json:"status"`
}

func (s *Service) CreatePayment(ctx context.Context, in CreatePaymentInput) (*CreatePaymentResult, error) {
	if in.OrderID == "" || in.UserID == "" {
		return nil, fmt.Errorf("%w: order_id and user_id required", ErrInvalidArgument)
	}
	if in.AmountFen <= 0 {
		return nil, fmt.Errorf("%w: amount_fen must > 0", ErrInvalidArgument)
	}

	if existing, err := s.repo.GetByOrderID(ctx, in.OrderID); err == nil {
		return toResult(existing), nil
	} else if !repository.IsNotFound(err) {
		return nil, err
	}

	payOrderID := idgen.New("pay")
	o := &model.PayOrder{
		PayOrderID: payOrderID,
		OrderID:    in.OrderID,
		UserID:     in.UserID,
		AmountFen:  in.AmountFen,
		Subject:    in.Subject,
		Status:     model.PayStatusUnpaid,
		PayURL:     fmt.Sprintf("%s/pay/%s", s.publicBaseURL, payOrderID),
	}
	if err := s.repo.CreatePayOrder(ctx, o); err != nil {
		if existing, gerr := s.repo.GetByOrderID(ctx, in.OrderID); gerr == nil {
			return toResult(existing), nil
		}
		return nil, err
	}
	return toResult(o), nil
}

type CallbackInput struct {
	CallbackID string
	PayOrderID string
	OrderID    string
	AmountFen  int64
	RawBody    string
}

type CallbackResult struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
	EventID  string `json:"event_id,omitempty"`
}

// HandleCallback 处理模拟支付回调：幂等落库 → 更新支付单 → 发 Kafka。
func (s *Service) HandleCallback(ctx context.Context, in CallbackInput) (*CallbackResult, error) {
	if in.CallbackID == "" {
		in.CallbackID = idgen.New("cb")
	}
	if in.PayOrderID == "" && in.OrderID == "" {
		return nil, fmt.Errorf("%w: pay_order_id or order_id required", ErrInvalidArgument)
	}

	cb := &model.PayCallback{
		CallbackID: in.CallbackID,
		PayOrderID: in.PayOrderID,
		OrderID:    in.OrderID,
		AmountFen:  in.AmountFen,
		RawBody:    in.RawBody,
		CreatedAt:  time.Now(),
	}
	inserted, err := s.repo.TryInsertCallback(ctx, cb)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return &CallbackResult{Accepted: true, Message: "duplicate callback ignored"}, nil
	}

	var (
		eventID string
		ev      kafka.PayEvent
		needPub bool
	)

	err = s.repo.WithTx(ctx, func(tx *repository.Repository) error {
		pay, gerr := loadPayOrder(ctx, tx, in.PayOrderID, in.OrderID)
		if gerr != nil {
			return gerr
		}
		if in.AmountFen > 0 && in.AmountFen != pay.AmountFen {
			return ErrAmountMismatch
		}

		cb.PayOrderID = pay.PayOrderID
		cb.OrderID = pay.OrderID
		cb.AmountFen = pay.AmountFen

		if pay.Status == model.PayStatusPaid {
			cb.ProcessResult = "already_paid"
			return tx.UpdateCallback(ctx, cb)
		}

		now := time.Now()
		pay.Status = model.PayStatusPaid
		pay.PaidAt = &now
		if err := tx.SavePayOrder(ctx, pay); err != nil {
			return err
		}

		eventID = idgen.New("evt")
		ev = kafka.PayEvent{
			EventID:    eventID,
			PayOrderID: pay.PayOrderID,
			OrderID:    pay.OrderID,
			AmountFen:  pay.AmountFen,
			PayStatus:  "success",
			PaidAtUnix: now.Unix(),
		}
		payload, _ := json.Marshal(ev)
		if err := tx.InsertOutbox(ctx, &model.PayEventOutbox{
			EventID:    eventID,
			OrderID:    pay.OrderID,
			PayOrderID: pay.PayOrderID,
			Payload:    string(payload),
			CreatedAt:  now,
		}); err != nil {
			return err
		}
		cb.ProcessResult = "paid"
		if err := tx.UpdateCallback(ctx, cb); err != nil {
			return err
		}
		needPub = true
		return nil
	})
	if errors.Is(err, ErrAmountMismatch) {
		return &CallbackResult{Accepted: false, Message: "amount mismatch"}, err
	}
	if errors.Is(err, ErrNotFound) {
		return &CallbackResult{Accepted: false, Message: "pay order not found"}, err
	}
	if err != nil {
		return nil, err
	}
	if needPub {
		if err := s.producer.PublishPaySuccess(ctx, ev); err != nil {
			return nil, err
		}
	}
	return &CallbackResult{Accepted: true, Message: "ok", EventID: eventID}, nil
}

func loadPayOrder(ctx context.Context, tx *repository.Repository, payOrderID, orderID string) (*model.PayOrder, error) {
	if payOrderID != "" {
		pay, err := tx.LockByPayOrderID(ctx, payOrderID)
		if repository.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return pay, err
	}
	pay, err := tx.GetByOrderID(ctx, orderID)
	if repository.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return tx.LockByPayOrderID(ctx, pay.PayOrderID)
}

func (s *Service) GetPayOrder(ctx context.Context, payOrderID string) (*model.PayOrder, error) {
	o, err := s.repo.GetByPayOrderID(ctx, payOrderID)
	if repository.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return o, err
}

// SimulatePay 模拟支付页一键支付。
func (s *Service) SimulatePay(ctx context.Context, payOrderID string) (*CallbackResult, error) {
	o, err := s.GetPayOrder(ctx, payOrderID)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]any{
		"pay_order_id": o.PayOrderID,
		"order_id":     o.OrderID,
		"amount_fen":   o.AmountFen,
		"source":       "mock_page",
	})
	return s.HandleCallback(ctx, CallbackInput{
		CallbackID: idgen.New("cb"),
		PayOrderID: o.PayOrderID,
		OrderID:    o.OrderID,
		AmountFen:  o.AmountFen,
		RawBody:    string(body),
	})
}

func toResult(o *model.PayOrder) *CreatePaymentResult {
	return &CreatePaymentResult{
		PayOrderID: o.PayOrderID,
		OrderID:    o.OrderID,
		AmountFen:  o.AmountFen,
		PayURL:     o.PayURL,
		Status:     o.Status,
	}
}
