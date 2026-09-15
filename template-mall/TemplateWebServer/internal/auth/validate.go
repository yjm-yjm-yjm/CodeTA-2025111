package auth

import (
	"errors"
	"regexp"
	"unicode/utf8"
)

var (
	ErrInvalidPhone      = errors.New("invalid phone format, expect 11-digit China mobile")
	ErrWeakPassword      = errors.New("password must be 6-64 characters")
	phonePattern         = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// NormalizePhone 校验并返回规范化手机号（仅数字）。
func NormalizePhone(phone string) (string, error) {
	p := trimASCII(phone)
	if !phonePattern.MatchString(p) {
		return "", ErrInvalidPhone
	}
	return p, nil
}

// ValidatePassword 登录/注册共用密码长度约束。
func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < 6 || n > 64 {
		return ErrWeakPassword
	}
	return nil
}

func trimASCII(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
