package ossstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrObjectNotFound  = errors.New("object not found")
	ErrObjectTooLarge  = errors.New("object exceeds size limit")
	ErrBadObjectKey    = errors.New("illegal object key")
)

// MaxFileSize 作业要求：单个模板文件最大 5 MB
const MaxFileSize = 5 * 1024 * 1024

var allowedExt = map[string]bool{
	".ppt": true, ".pptx": true, ".doc": true, ".docx": true,
	".xls": true, ".xlsx": true, ".pdf": true,
	// 封面预览图
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
}

type Credential struct {
	Provider    string `json:"provider"`
	Bucket      string `json:"bucket"`
	ObjectKey   string `json:"object_key"`
	UploadURL   string `json:"upload_url"`
	Method      string `json:"method"` // PUT
	ExpireAt    int64  `json:"expire_at_unix"`
	AccessKeyID string `json:"access_key_id,omitempty"`
	Policy      string `json:"policy,omitempty"`
	Signature   string `json:"signature,omitempty"`
	ContentType string `json:"content_type"`
}

type ObjectMeta struct {
	Bucket   string
	Key      string
	Size     int64
	Exists   bool
}

type Store interface {
	IssueUpload(ctx context.Context, originalFilename, contentType string) (*Credential, error)
	Confirm(ctx context.Context, objectKey string, expectSize int64) (*ObjectMeta, error)
	GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error)
	PutObject(ctx context.Context, objectKey, contentType string, body io.Reader, size int64) error
	NewObjectKey(ext string) string
	ProviderName() string
	BucketName() string
}

func New(provider, endpoint, ak, sk, bucket, publicBase string, ttl time.Duration) (Store, error) {
	switch strings.ToLower(provider) {
	case "aliyun":
		return NewAliyun(endpoint, ak, sk, bucket, publicBase, ttl)
	default:
		return NewMock(bucket, publicBase, ttl), nil
	}
}

func normalizeFilename(name string) (ext string, err error) {
	name = path.Base(strings.TrimSpace(name))
	ext = strings.ToLower(path.Ext(name))
	if !allowedExt[ext] {
		return "", fmt.Errorf("%w: unsupported file type %s", ErrInvalidArgument, ext)
	}
	return ext, nil
}

func newObjectKey(ext string) string {
	day := time.Now().Format("20060102")
	return fmt.Sprintf("templates/%s/%s%s", day, uuid.NewString(), ext)
}

// -------- Mock --------

type MockStore struct {
	bucket string
	base   string
	ttl    time.Duration
	mu     sync.Mutex
	// issued keys -> expire
	issued map[string]time.Time
	// uploaded objects size
	objects map[string]int64
}

func NewMock(bucket, publicBase string, ttl time.Duration) *MockStore {
	return &MockStore{
		bucket:  bucket,
		base:    strings.TrimRight(publicBase, "/"),
		ttl:     ttl,
		issued:  map[string]time.Time{},
		objects: map[string]int64{},
	}
}

func (m *MockStore) ProviderName() string { return "mock" }
func (m *MockStore) BucketName() string   { return m.bucket }
func (m *MockStore) NewObjectKey(ext string) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return newObjectKey(strings.ToLower(ext))
}

func (m *MockStore) GetObject(_ context.Context, objectKey string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objects[objectKey]; !ok {
		return nil, ErrObjectNotFound
	}
	// mock 不存正文，返回空体即可（封面抽取联调走 aliyun）
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (m *MockStore) PutObject(_ context.Context, objectKey, contentType string, body io.Reader, size int64) error {
	_ = contentType
	if size <= 0 {
		n, err := io.Copy(io.Discard, body)
		if err != nil {
			return err
		}
		size = n
	} else {
		_, _ = io.CopyN(io.Discard, body, size)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issued[objectKey] = time.Now().Add(m.ttl)
	m.objects[objectKey] = size
	return nil
}

func (m *MockStore) IssueUpload(_ context.Context, originalFilename, contentType string) (*Credential, error) {
	ext, err := normalizeFilename(originalFilename)
	if err != nil {
		return nil, err
	}
	key := newObjectKey(ext)
	exp := time.Now().Add(m.ttl)
	m.mu.Lock()
	m.issued[key] = exp
	m.mu.Unlock()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &Credential{
		Provider:    "mock",
		Bucket:      m.bucket,
		ObjectKey:   key,
		UploadURL:   fmt.Sprintf("%s/api/upload/mock?key=%s", m.base, key),
		Method:      "PUT",
		ExpireAt:    exp.Unix(),
		ContentType: contentType,
	}, nil
}

func (m *MockStore) PutMockObject(key string, size int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.issued[key]
	if !ok || time.Now().After(exp) {
		return ErrBadObjectKey
	}
	if size <= 0 || size > MaxFileSize {
		return ErrObjectTooLarge
	}
	m.objects[key] = size
	return nil
}

func (m *MockStore) Confirm(_ context.Context, objectKey string, expectSize int64) (*ObjectMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.issued[objectKey]; !ok {
		return nil, ErrBadObjectKey
	}
	size, ok := m.objects[objectKey]
	if !ok {
		return nil, ErrObjectNotFound
	}
	if size > MaxFileSize {
		return nil, ErrObjectTooLarge
	}
	if expectSize > 0 && expectSize != size {
		return nil, fmt.Errorf("%w: size mismatch", ErrInvalidArgument)
	}
	return &ObjectMeta{Bucket: m.bucket, Key: objectKey, Size: size, Exists: true}, nil
}

// -------- Aliyun --------

type AliyunStore struct {
	client     *oss.Client
	bucket     *oss.Bucket
	ak         string
	sk         string
	bucketName string
	endpoint   string
	ttl        time.Duration
	mu         sync.Mutex
	issued     map[string]time.Time
}

func NewAliyun(endpoint, ak, sk, bucket, publicBase string, ttl time.Duration) (*AliyunStore, error) {
	_ = publicBase
	if endpoint == "" || ak == "" || sk == "" || bucket == "" {
		return nil, fmt.Errorf("%w: aliyun oss config incomplete", ErrInvalidArgument)
	}
	cli, err := oss.New(endpoint, ak, sk)
	if err != nil {
		return nil, err
	}
	b, err := cli.Bucket(bucket)
	if err != nil {
		return nil, err
	}
	s := &AliyunStore{
		client: cli, bucket: b, ak: ak, sk: sk,
		bucketName: bucket, endpoint: endpoint, ttl: ttl,
		issued: map[string]time.Time{},
	}
	// 签名直传依赖浏览器跨域 PUT：启动时写入桶 CORS（失败不阻断，便于本地排查）
	if err := s.EnsureBrowserCORS(); err != nil {
		fmt.Printf("oss cors ensure warning: %v\n", err)
	}
	return s, nil
}

// EnsureBrowserCORS 允许管理端前端对预签名 URL 发起 PUT/GET（签名直传）。
func (a *AliyunStore) EnsureBrowserCORS() error {
	rule := oss.CORSRule{
		// 作业本地联调：管理端 Vite 常见来源；生产可收紧为正式域名
		AllowedOrigin: []string{
			"http://localhost:3001",
			"http://127.0.0.1:3001",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		},
		// 阿里云 AllowedMethod 仅支持 GET/PUT/POST/DELETE/HEAD，不要写 OPTIONS
		AllowedMethod: []string{"GET", "PUT", "POST", "HEAD"},
		AllowedHeader: []string{"*"},
		ExposeHeader:  []string{"ETag", "x-oss-request-id", "x-oss-version-id"},
		MaxAgeSeconds: 600,
	}
	return a.client.SetBucketCORS(a.bucketName, []oss.CORSRule{rule})
}

func (a *AliyunStore) ProviderName() string { return "aliyun" }
func (a *AliyunStore) BucketName() string   { return a.bucketName }
func (a *AliyunStore) NewObjectKey(ext string) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return newObjectKey(strings.ToLower(ext))
}

func (a *AliyunStore) GetObject(_ context.Context, objectKey string) (io.ReadCloser, error) {
	rc, err := a.bucket.GetObject(objectKey)
	if err != nil {
		if ossErr, ok := err.(oss.ServiceError); ok && ossErr.StatusCode == http.StatusNotFound {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	return rc, nil
}

func (a *AliyunStore) PutObject(_ context.Context, objectKey, contentType string, body io.Reader, size int64) error {
	opts := []oss.Option{}
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	if size > 0 {
		opts = append(opts, oss.ContentLength(size))
	}
	return a.bucket.PutObject(objectKey, body, opts...)
}

func (a *AliyunStore) IssueUpload(_ context.Context, originalFilename, contentType string) (*Credential, error) {
	ext, err := normalizeFilename(originalFilename)
	if err != nil {
		return nil, err
	}
	key := newObjectKey(ext)
	exp := time.Now().Add(a.ttl)
	a.mu.Lock()
	a.issued[key] = exp
	a.mu.Unlock()

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// 签名直传：服务端只签发短期 PUT URL，文件数据由浏览器直传 OSS，不经本服务转发。
	signed, err := a.bucket.SignURL(key, oss.HTTPPut, int64(a.ttl.Seconds()), oss.ContentType(contentType))
	if err != nil {
		return nil, err
	}
	return &Credential{
		Provider:    "aliyun",
		Bucket:      a.bucketName,
		ObjectKey:   key,
		UploadURL:   signed,
		Method:      "PUT",
		ExpireAt:    exp.Unix(),
		ContentType: contentType,
	}, nil
}

func (a *AliyunStore) Confirm(ctx context.Context, objectKey string, expectSize int64) (*ObjectMeta, error) {
	_ = ctx
	a.mu.Lock()
	_, issued := a.issued[objectKey]
	a.mu.Unlock()
	if !issued {
		// 允许重启后仍确认：校验 key 前缀即可
		if !strings.HasPrefix(objectKey, "templates/") {
			return nil, ErrBadObjectKey
		}
	}
	meta, err := a.bucket.GetObjectMeta(objectKey)
	if err != nil {
		if ossErr, ok := err.(oss.ServiceError); ok && ossErr.StatusCode == http.StatusNotFound {
			return nil, ErrObjectNotFound
		}
		// Head via IsObjectExist
		exist, e2 := a.bucket.IsObjectExist(objectKey)
		if e2 != nil {
			return nil, err
		}
		if !exist {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	size := parseSize(meta)
	if size > MaxFileSize {
		return nil, ErrObjectTooLarge
	}
	if expectSize > 0 && expectSize != size {
		return nil, fmt.Errorf("%w: size mismatch", ErrInvalidArgument)
	}
	return &ObjectMeta{Bucket: a.bucketName, Key: objectKey, Size: size, Exists: true}, nil
}

func parseSize(h http.Header) int64 {
	v := h.Get("Content-Length")
	if v == "" {
		return 0
	}
	var n int64
	_, _ = fmt.Sscan(v, &n)
	return n
}
