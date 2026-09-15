package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type PayEvent struct {
	EventID    string `json:"event_id"`
	PayOrderID string `json:"pay_order_id"`
	OrderID    string `json:"order_id"`
	AmountFen  int64  `json:"amount_fen"`
	PayStatus  string `json:"pay_status"`
	PaidAtUnix int64  `json:"paid_at_unix"`
}

type Producer interface {
	PublishPaySuccess(ctx context.Context, ev PayEvent) error
	Close() error
}

type WriterProducer struct {
	w       *kafka.Writer
	timeout time.Duration
}

func NewProducer(brokers []string, topic string, timeout time.Duration) *WriterProducer {
	return &WriterProducer{
		timeout: timeout,
		w: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
	}
}

func (p *WriterProducer) Close() error {
	return p.w.Close()
}

func (p *WriterProducer) PublishPaySuccess(ctx context.Context, ev PayEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	err = p.w.WriteMessages(cctx, kafka.Message{
		Key:   []byte(ev.OrderID),
		Value: body,
		Time:  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

// NoopProducer 单测用。
type NoopProducer struct{}

func (NoopProducer) PublishPaySuccess(context.Context, PayEvent) error { return nil }
func (NoopProducer) Close() error                                      { return nil }
