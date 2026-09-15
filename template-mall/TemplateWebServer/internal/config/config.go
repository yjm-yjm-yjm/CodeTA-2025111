package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr string

	OrderGRPCAddr string
	GRPCTimeout   time.Duration

	JWTSecret string
	JWTTTL    time.Duration

	// AuthUsersFile 本地账号文件（BFF 不接业务库，仅存注册登录凭证）
	AuthUsersFile string
}

func Load() Config {
	loadDotEnv()
	return Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		OrderGRPCAddr: getEnv("ORDER_GRPC_ADDR", "127.0.0.1:50051"),
		GRPCTimeout:   getDuration("GRPC_TIMEOUT", 5*time.Second),
		JWTSecret:     getEnv("JWT_SECRET", "dev-c-end-jwt-secret-change-me"),
		JWTTTL:        getDuration("JWT_TTL", 24*time.Hour),
		AuthUsersFile: getEnv("AUTH_USERS_FILE", "./data/users.json"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func getDuration(k string, d time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	if x, err := time.ParseDuration(v); err == nil {
		return x
	}
	if sec, err := strconv.Atoi(v); err == nil {
		return time.Duration(sec) * time.Second
	}
	return d
}
