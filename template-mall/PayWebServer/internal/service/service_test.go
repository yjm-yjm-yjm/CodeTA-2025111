package service_test

import (
	"context"
	"sync"
	"testing"

	"template-mall/PayWebServer/internal/kafka"
	"template-mall/PayWebServer/internal/repository"
	"template-mall/PayWebServer/internal/service"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type memProducer struct {
	mu   sync.Mutex
	evts []kafka.PayEvent
}

func (m *memProducer) PublishPaySuccess(_ context.Context, ev kafka.PayEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evts = append(m.evts, ev)
	return nil
}
func (m *memProducer) Close() error { return nil }

func setup(t *testing.T) (*service.Service, *memProducer) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.New(db)
	if err := repo.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	prod := &memProducer{}
	return service.New(repo, prod, "http://127.0.0.1:8082"), prod
}

func TestCreatePaymentReuse(t *testing.T) {
	svc, _ := setup(t)
	r1, err := svc.CreatePayment(context.Background(), service.CreatePaymentInput{
		OrderID: "ord1", AmountFen: 100, UserID: "u1", Subject: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := svc.CreatePayment(context.Background(), service.CreatePaymentInput{
		OrderID: "ord1", AmountFen: 100, UserID: "u1", Subject: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	if r1.PayOrderID != r2.PayOrderID {
		t.Fatalf("want reuse pay order")
	}
}

func TestCreatePaymentConcurrent(t *testing.T) {
	svc, _ := setup(t)
	const n = 16
	ids := make(chan string, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			r, err := svc.CreatePayment(context.Background(), service.CreatePaymentInput{
				OrderID: "ord_c", AmountFen: 200, UserID: "u1", Subject: "t",
			})
			if err != nil {
				t.Errorf("%v", err)
				return
			}
			ids <- r.PayOrderID
		}()
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		} else if id != first {
			t.Fatalf("diverged %s vs %s", first, id)
		}
	}
}

func TestCallbackIdempotent(t *testing.T) {
	svc, prod := setup(t)
	created, _ := svc.CreatePayment(context.Background(), service.CreatePaymentInput{
		OrderID: "ord2", AmountFen: 300, UserID: "u1", Subject: "t",
	})
	in := service.CallbackInput{
		CallbackID: "cb-same",
		PayOrderID: created.PayOrderID,
		OrderID:    created.OrderID,
		AmountFen:  300,
		RawBody:    "{}",
	}
	if _, err := svc.HandleCallback(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.HandleCallback(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	prod.mu.Lock()
	defer prod.mu.Unlock()
	if len(prod.evts) != 1 {
		t.Fatalf("kafka events=%d want 1", len(prod.evts))
	}
	if prod.evts[0].OrderID != "ord2" || prod.evts[0].PayStatus != "success" {
		t.Fatalf("bad event %+v", prod.evts[0])
	}
}

func TestCallbackAmountMismatch(t *testing.T) {
	svc, _ := setup(t)
	created, _ := svc.CreatePayment(context.Background(), service.CreatePaymentInput{
		OrderID: "ord3", AmountFen: 400, UserID: "u1", Subject: "t",
	})
	_, err := svc.HandleCallback(context.Background(), service.CallbackInput{
		CallbackID: "cb-amt",
		PayOrderID: created.PayOrderID,
		AmountFen:  1,
		RawBody:    "{}",
	})
	if err != service.ErrAmountMismatch {
		t.Fatalf("want amount mismatch, got %v", err)
	}
}

func TestSimulatePay(t *testing.T) {
	svc, prod := setup(t)
	created, _ := svc.CreatePayment(context.Background(), service.CreatePaymentInput{
		OrderID: "ord4", AmountFen: 500, UserID: "u1", Subject: "demo",
	})
	if _, err := svc.SimulatePay(context.Background(), created.PayOrderID); err != nil {
		t.Fatal(err)
	}
	o, err := svc.GetPayOrder(context.Background(), created.PayOrderID)
	if err != nil || o.Status != "paid" {
		t.Fatalf("want paid, got %+v err=%v", o, err)
	}
	if len(prod.evts) != 1 {
		t.Fatalf("events=%d", len(prod.evts))
	}
}
