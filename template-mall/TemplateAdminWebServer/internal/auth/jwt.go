package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrUnauthorized = errors.New("unauthorized")

type Claims struct {
	AdminID  string `json:"aid"`
	Nickname string `json:"nick"`
	Source   string `json:"src"` // wps | mock
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl}
}

func (t *TokenIssuer) Issue(adminID, nickname, source string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		AdminID:  adminID,
		Nickname: nickname,
		Source:   source,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	})
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
