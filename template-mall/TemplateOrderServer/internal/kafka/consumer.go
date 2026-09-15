package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"template-mall/TemplateOrderServer/internal/service"

	"github.com/segmentio/kafka-go"
)

// PayEvent 与 PayWebServer 约定的消息结构。
type PayEvent struct {
	EventID    string `json:"event_id"`
	PayOrderID string `json:"pay_order_id"`
	OrderID    string `json:"order_id"`
	AmountFen  int64  `json:"amount_fen"`
	PayStatus  string `json:"pay_status"`
	PaidAtUnix int64  `json:"paid_at_unix"`
}

type Consumer struct {
	reader *kafka.Reader
	svc    *service.Service
}

func NewConsumer(brokers []string, topic, groupID string, svc *service.Service) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // 手动提交：业务成功后再 commit
		StartOffset:    kafka.FirstOffset,
	})
	return &Consumer{reader: r, svc: svc}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) Run(ctx context.Context) {
	log.Printf("kafka consumer started topic=%s group=%s", c.reader.Config().Topic, c.reader.Config().GroupID)
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if err := c.handle(ctx, msg); err != nil {
			log.Printf("kafka handle error key=%s: %v", string(msg.Key), err)
			// 不 commit，等待重投；作业不要求死信
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("kafka commit error: %v", err)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, msg kafka.Message) error {
	var ev PayEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		log.Printf("invalid pay event payload: %s", string(msg.Value))
		// 坏消息直接跳过并允许 commit，避免毒丸堵队列
		return nil
	}
	if ev.PayStatus != "success" && ev.PayStatus != "SUCCESS" && ev.PayStatus != "paid" {
		log.Printf("ignore non-success pay event status=%s event_id=%s", ev.PayStatus, ev.EventID)
		return nil
	}
	err := c.svc.HandlePaySuccess(ctx, ev.EventID, ev.OrderID, ev.PayOrderID, ev.AmountFen)
	if errors.Is(err, service.ErrOrderCancelled) {
		log.Printf("ERROR: pay success for cancelled order order_id=%s event_id=%s", ev.OrderID, ev.EventID)
		return nil // 记录错误后视为已处理，避免反复刷
	}
	if errors.Is(err, service.ErrAmountMismatch) {
		log.Printf("ERROR: amount mismatch order_id=%s event_id=%s amount=%d", ev.OrderID, ev.EventID, ev.AmountFen)
		return nil
	}
	return err
}
