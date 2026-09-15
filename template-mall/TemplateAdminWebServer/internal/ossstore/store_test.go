package ossstore_test

import (
	"context"
	"testing"
	"time"

	"template-mall/TemplateAdminWebServer/internal/ossstore"
)

func TestMockUploadConfirm(t *testing.T) {
	s := ossstore.NewMock("bkt", "http://127.0.0.1:8081", time.Minute)
	cred, err := s.IssueUpload(context.Background(), "demo.pptx", "application/vnd.openxmlformats")
	if err != nil {
		t.Fatal(err)
	}
	if cred.ObjectKey == "" || cred.UploadURL == "" {
		t.Fatalf("bad cred %+v", cred)
	}
	if err := s.PutMockObject(cred.ObjectKey, 1024); err != nil {
		t.Fatal(err)
	}
	meta, err := s.Confirm(context.Background(), cred.ObjectKey, 1024)
	if err != nil || !meta.Exists || meta.Size != 1024 {
		t.Fatalf("confirm: %+v err=%v", meta, err)
	}
}

func TestRejectBadExtAndKey(t *testing.T) {
	s := ossstore.NewMock("bkt", "http://127.0.0.1:8081", time.Minute)
	if _, err := s.IssueUpload(context.Background(), "a.exe", ""); err == nil {
		t.Fatal("want reject exe")
	}
	if err := s.PutMockObject("not-issued", 10); err != ossstore.ErrBadObjectKey {
		t.Fatalf("want bad key, got %v", err)
	}
	cred, _ := s.IssueUpload(context.Background(), "a.docx", "")
	if err := s.PutMockObject(cred.ObjectKey, ossstore.MaxFileSize+1); err != ossstore.ErrObjectTooLarge {
		t.Fatalf("want too large, got %v", err)
	}
}
