package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredential = errors.New("invalid phone or password")
	ErrUserExists        = errors.New("phone already registered")
	ErrUnauthorized      = errors.New("unauthorized")
)

type UserRecord struct {
	UserID       string `json:"user_id"`
	// Username 字段存手机号（兼容旧 users.json 的 username 键）。
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	PasswordHash string `json:"password_hash"`
	CreatedAt    int64  `json:"created_at"`
}

// Phone 返回登录手机号。
func (u *UserRecord) Phone() string { return u.Username }

type Store struct {
	mu   sync.Mutex
	path string
	byName map[string]*UserRecord
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, byName: map[string]*UserRecord{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.persistLocked()
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}
	var list []*UserRecord
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	s.byName = map[string]*UserRecord{}
	for _, u := range list {
		s.byName[u.Username] = u
	}
	return nil
}

func (s *Store) persistLocked() error {
	list := make([]*UserRecord, 0, len(s.byName))
	for _, u := range s.byName {
		list = append(list, u)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Register(phone, password, nickname string) (*UserRecord, error) {
	phone, err := NormalizePhone(phone)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byName[phone]; ok {
		return nil, ErrUserExists
	}
	if nickname == "" {
		nickname = phone
	}
	u := &UserRecord{
		UserID:       uuid.NewString(),
		Username:     phone,
		Nickname:     nickname,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().Unix(),
	}
	s.byName[phone] = u
	if err := s.persistLocked(); err != nil {
		delete(s.byName, phone)
		return nil, err
	}
	return u, nil
}

func (s *Store) Authenticate(phone, password string) (*UserRecord, error) {
	phone, err := NormalizePhone(phone)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, ErrInvalidCredential
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byName[phone]
	if !ok {
		return nil, ErrInvalidCredential
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredential
	}
	cp := *u
	return &cp, nil
}

func (s *Store) GetByUserID(userID string) (*UserRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.byName {
		if u.UserID == userID {
			cp := *u
			return &cp, true
		}
	}
	return nil, false
}

type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"uname"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl}
}

func (t *TokenIssuer) Issue(userID, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

func (t *TokenIssuer) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnauthorized
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, ErrUnauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrUnauthorized
	}
	return claims, nil
}
