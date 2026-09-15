package auth_test

import (
	"path/filepath"
	"testing"
	"time"

	"template-mall/TemplateWebServer/internal/auth"
)

func TestRegisterLoginJWT(t *testing.T) {
	dir := t.TempDir()
	store, err := auth.NewStore(filepath.Join(dir, "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	u, err := store.Register("13800138000", "secret123", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Register("13800138000", "secret123", ""); err != auth.ErrUserExists {
		t.Fatalf("want exists, got %v", err)
	}
	if _, err := store.Register("12345", "secret123", ""); err != auth.ErrInvalidPhone {
		t.Fatalf("want invalid phone, got %v", err)
	}
	if _, err := store.Register("13900139000", "123", ""); err != auth.ErrWeakPassword {
		t.Fatalf("want weak password, got %v", err)
	}
	got, err := store.Authenticate("13800138000", "secret123")
	if err != nil || got.UserID != u.UserID {
		t.Fatalf("auth failed: %v", err)
	}
	if _, err := store.Authenticate("13800138000", "badbad"); err != auth.ErrInvalidCredential {
		t.Fatalf("want bad credential")
	}

	issuer := auth.NewTokenIssuer("test-secret", time.Hour)
	tok, err := issuer.Issue(u.UserID, u.Phone())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Parse(tok)
	if err != nil || claims.UserID != u.UserID {
		t.Fatalf("parse: %v %+v", err, claims)
	}
}
