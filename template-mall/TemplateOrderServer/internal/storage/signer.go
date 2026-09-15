package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type PresignResult struct {
	URL      string
	ExpireAt time.Time
}

type Signer interface {
	PresignGet(ctx context.Context, bucket, objectKey string, ttl time.Duration) (PresignResult, error)
}

type Config struct {
	Provider   string
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	MockSecret string
}

// MockSigner 本地开发用短期鉴权 URL（非真实 OSS）。
type MockSigner struct {
	Secret string
	Base   string
}

func (m MockSigner) PresignGet(_ context.Context, bucket, objectKey string, ttl time.Duration) (PresignResult, error) {
	exp := time.Now().Add(ttl).Unix()
	msg := fmt.Sprintf("%s\n%s\n%d", bucket, objectKey, exp)
	mac := hmac.New(sha256.New, []byte(m.Secret))
	_, _ = mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))
	base := m.Base
	if base == "" {
		base = "http://127.0.0.1:18080/mock-oss"
	}
	u, _ := url.Parse(base + "/download")
	q := u.Query()
	q.Set("bucket", bucket)
	q.Set("key", objectKey)
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", sig)
	u.RawQuery = q.Encode()
	return PresignResult{URL: u.String(), ExpireAt: time.Unix(exp, 0)}, nil
}

type AliyunSigner struct {
	client *oss.Client
}

func NewAliyunSigner(endpoint, ak, sk string) (*AliyunSigner, error) {
	if endpoint == "" || ak == "" || sk == "" {
		return nil, fmt.Errorf("aliyun oss config incomplete")
	}
	cli, err := oss.New(endpoint, ak, sk)
	if err != nil {
		return nil, err
	}
	return &AliyunSigner{client: cli}, nil
}

func (a *AliyunSigner) PresignGet(_ context.Context, bucket, objectKey string, ttl time.Duration) (PresignResult, error) {
	b, err := a.client.Bucket(bucket)
	if err != nil {
		return PresignResult{}, err
	}
	sec := int64(ttl.Seconds())
	if sec <= 0 {
		sec = 600
	}
	signed, err := b.SignURL(objectKey, oss.HTTPGet, sec)
	if err != nil {
		return PresignResult{}, err
	}
	return PresignResult{
		URL:      signed,
		ExpireAt: time.Now().Add(time.Duration(sec) * time.Second),
	}, nil
}

func NewSigner(cfg Config) (Signer, error) {
	switch strings.ToLower(cfg.Provider) {
	case "aliyun":
		return NewAliyunSigner(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
	default:
		return MockSigner{Secret: cfg.MockSecret}, nil
	}
}
