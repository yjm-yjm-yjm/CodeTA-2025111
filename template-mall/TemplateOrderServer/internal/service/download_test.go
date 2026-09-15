package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"template-mall/TemplateOrderServer/internal/model"
	"template-mall/TemplateOrderServer/internal/payclient"
	"template-mall/TemplateOrderServer/internal/repository"
	"template-mall/TemplateOrderServer/internal/service"
	"template-mall/TemplateOrderServer/internal/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type stubPay struct {
	mu    sync.Mutex
	calls int
	byOrd map[string]*payclient.CreatePaymentResponse
}

func (s *stubPay) CreatePayment(_ context.Context, req payclient.CreatePaymentRequest) (*payclient.CreatePaymentResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.byOrd == nil {
		s.byOrd = map[string]*payclient.CreatePaymentResponse{}
	}
	if existing, ok := s.byOrd[req.OrderID]; ok {
		return existing, nil
	}
	out := &payclient.CreatePaymentResponse{
		PayOrderID: fmt.Sprintf("pay_%s", req.OrderID),
		OrderID:    req.OrderID,
		AmountFen:  req.AmountFen,
		PayURL:     "http://pay.local/mock/" + req.OrderID,
		Status:     "unpaid",
	}
	s.byOrd[req.OrderID] = out
	return out, nil
}

func setupService(t *testing.T) (*service.Service, *stubPay) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	repo := repository.New(db)
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pay := &stubPay{}
	svc := service.New(repo, pay, storage.MockSigner{Secret: "test"}, time.Minute)
	return svc, pay
}

func createOnShelf(t *testing.T, svc *service.Service, name string, priceFen int64, priceType string) *model.Template {
	t.Helper()
	tpl, err := svc.CreateTemplate(context.Background(), service.CreateTemplateInput{
		Name: name, FileType: "pptx", PriceFen: priceFen, PriceType: priceType,
		StorageProvider: "mock", Bucket: "b", ObjectKey: "k/" + name, OriginalFilename: name + ".pptx",
		FileSize: 1024, Publish: true,
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	return tpl
}

func TestFreeDownloadIdempotent(t *testing.T) {
	svc, _ := setupService(t)
	tpl := createOnShelf(t, svc, "free1", 0, model.PriceTypeFree)

	r1, err := svc.DownloadTemplate(context.Background(), "u1", tpl.TemplateID)
	if err != nil || r1.Granted == nil {
		t.Fatalf("first download: err=%v res=%+v", err, r1)
	}
	r2, err := svc.DownloadTemplate(context.Background(), "u1", tpl.TemplateID)
	if err != nil || r2.Granted == nil {
		t.Fatalf("second download: err=%v", err)
	}
	if r1.Granted.Order.OrderID != r2.Granted.Order.OrderID {
		t.Fatalf("order id mismatch %s vs %s", r1.Granted.Order.OrderID, r2.Granted.Order.OrderID)
	}
}

func TestFreeDownloadConcurrent(t *testing.T) {
	svc, _ := setupService(t)
	tpl := createOnShelf(t, svc, "free2", 0, model.PriceTypeFree)

	const n = 20
	ids := make(chan string, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			r, err := svc.DownloadTemplate(context.Background(), "u2", tpl.TemplateID)
			if err != nil || r.Granted == nil {
				t.Errorf("download: %v", err)
				return
			}
			ids <- r.Granted.Order.OrderID
		}()
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		} else if id != first {
			t.Fatalf("concurrent free orders diverged: %s vs %s", first, id)
		}
	}
}

func TestOffShelfRejected(t *testing.T) {
	svc, _ := setupService(t)
	tpl, err := svc.CreateTemplate(context.Background(), service.CreateTemplateInput{
		Name: "draft", FileType: "docx", PriceFen: 0, PriceType: model.PriceTypeFree,
		StorageProvider: "mock", Bucket: "b", ObjectKey: "k/d", OriginalFilename: "d.docx",
		FileSize: 100, Publish: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.DownloadTemplate(context.Background(), "u1", tpl.TemplateID)
	if err != service.ErrTemplateOffShelf {
		t.Fatalf("want off shelf, got %v", err)
	}
}

func TestMemberDownload(t *testing.T) {
	svc, _ := setupService(t)
	tpl := createOnShelf(t, svc, "paid1", 1200, model.PriceTypePaid)
	if _, err := svc.SetMembership(context.Background(), "m1", true); err != nil {
		t.Fatal(err)
	}
	r, err := svc.DownloadTemplate(context.Background(), "m1", tpl.TemplateID)
	if err != nil || r.Granted == nil {
		t.Fatalf("member download: %v %+v", err, r)
	}
	if r.Granted.Order.OrderType != model.OrderTypeMember {
		t.Fatalf("want member order, got %s", r.Granted.Order.OrderType)
	}
}

func TestRetailNeedPayAndReuse(t *testing.T) {
	svc, pay := setupService(t)
	tpl := createOnShelf(t, svc, "paid2", 990, model.PriceTypePaid)

	r1, err := svc.DownloadTemplate(context.Background(), "n1", tpl.TemplateID)
	if err != nil || r1.NeedPay == nil {
		t.Fatalf("want payment required: %v %+v", err, r1)
	}
	r2, err := svc.DownloadTemplate(context.Background(), "n1", tpl.TemplateID)
	if err != nil || r2.NeedPay == nil {
		t.Fatalf("reuse: %v", err)
	}
	if r1.NeedPay.Order.OrderID != r2.NeedPay.Order.OrderID {
		t.Fatalf("order not reused")
	}
	if r1.NeedPay.Payment.PayOrderID != r2.NeedPay.Payment.PayOrderID {
		t.Fatalf("pay order not reused")
	}
	if pay.calls != 1 {
		t.Fatalf("pay create calls=%d want 1", pay.calls)
	}
}

func TestCancelThenRedownload(t *testing.T) {
	svc, _ := setupService(t)
	tpl := createOnShelf(t, svc, "paid3", 500, model.PriceTypePaid)
	r1, err := svc.DownloadTemplate(context.Background(), "n2", tpl.TemplateID)
	if err != nil || r1.NeedPay == nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelOrder(context.Background(), "n2", r1.NeedPay.Order.OrderID); err != nil {
		t.Fatal(err)
	}
	r2, err := svc.DownloadTemplate(context.Background(), "n2", tpl.TemplateID)
	if err != nil || r2.NeedPay == nil {
		t.Fatal(err)
	}
	if r1.NeedPay.Order.OrderID == r2.NeedPay.Order.OrderID {
		t.Fatalf("cancelled order should not be reused")
	}
}

func TestPaySuccessAndRedownload(t *testing.T) {
	svc, _ := setupService(t)
	tpl := createOnShelf(t, svc, "paid4", 800, model.PriceTypePaid)
	r1, err := svc.DownloadTemplate(context.Background(), "n3", tpl.TemplateID)
	if err != nil || r1.NeedPay == nil {
		t.Fatal(err)
	}
	oid := r1.NeedPay.Order.OrderID
	if err := svc.HandlePaySuccess(context.Background(), "evt1", oid, r1.NeedPay.Payment.PayOrderID, 800); err != nil {
		t.Fatal(err)
	}
	// 重复事件幂等
	if err := svc.HandlePaySuccess(context.Background(), "evt1", oid, r1.NeedPay.Payment.PayOrderID, 800); err != nil {
		t.Fatal(err)
	}
	r2, err := svc.DownloadTemplate(context.Background(), "n3", tpl.TemplateID)
	if err != nil || r2.Granted == nil {
		t.Fatalf("after pay: %v %+v", err, r2)
	}
	if r2.Granted.Order.OrderID != oid {
		t.Fatalf("should reuse retail paid order")
	}
}

func TestCancelledOrderIgnoresPay(t *testing.T) {
	svc, _ := setupService(t)
	tpl := createOnShelf(t, svc, "paid5", 700, model.PriceTypePaid)
	r1, _ := svc.DownloadTemplate(context.Background(), "n4", tpl.TemplateID)
	oid := r1.NeedPay.Order.OrderID
	if _, err := svc.CancelOrder(context.Background(), "n4", oid); err != nil {
		t.Fatal(err)
	}
	err := svc.HandlePaySuccess(context.Background(), "evt2", oid, "payx", 700)
	if err != service.ErrOrderCancelled {
		t.Fatalf("want cancelled, got %v", err)
	}
}
