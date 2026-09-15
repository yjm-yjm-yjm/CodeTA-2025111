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

	// AuthMode: mock | wps
	AuthMode string

	// WPS OAuth
	WPSClientID     string
	WPSClientSecret string
	WPSRedirectURI  string
	WPSAuthURL      string
	WPSTokenURL     string
	WPSUserInfoURL  string
	WPSScopes       string
	FrontendURL     string // 回调成功后跳转管理端前端，默认 http://localhost:3001

	// OSS
	OSSProvider   string // mock | aliyun
	OSSEndpoint   string
	OSSAccessKey  string
	OSSSecretKey  string
	OSSBucket     string
	OSSUploadTTL  time.Duration
	PublicBaseURL string // 本服务对外地址，用于 mock 上传 URL
}

func Load() Config {
	loadDotEnv()
	return Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8081"),
		OrderGRPCAddr: getEnv("ORDER_GRPC_ADDR", "127.0.0.1:50051"),
		GRPCTimeout:   getDuration("GRPC_TIMEOUT", 5*time.Second),
		JWTSecret:     getEnv("JWT_SECRET", "dev-admin-jwt-secret-change-me"),
		JWTTTL:        getDuration("JWT_TTL", 12*time.Hour),

		AuthMode:        getEnv("AUTH_MODE", "mock"),
		WPSClientID:     getEnv("WPS_CLIENT_ID", ""),
		WPSClientSecret: getEnv("WPS_CLIENT_SECRET", ""),
		WPSRedirectURI:  getEnv("WPS_REDIRECT_URI", "http://localhost:3001/api/auth/callback"),
		WPSAuthURL:      getEnv("WPS_AUTH_URL", "https://openapi.wps.cn/oauth2/auth"),
		WPSTokenURL:     getEnv("WPS_TOKEN_URL", "https://openapi.wps.cn/oauth2/token"),
		WPSUserInfoURL:  getEnv("WPS_USERINFO_URL", "https://openapi.wps.cn/v7/users/current"),
		WPSScopes:       getEnv("WPS_SCOPES", "kso.user_base.read"),
		FrontendURL:     getEnv("ADMIN_FRONTEND_URL", "http://localhost:3001"),

		OSSProvider:   getEnv("OSS_PROVIDER", "mock"),
		OSSEndpoint:   getEnv("OSS_ENDPOINT", ""),
		OSSAccessKey:  getEnv("OSS_ACCESS_KEY", ""),
		OSSSecretKey:  getEnv("OSS_SECRET_KEY", ""),
		OSSBucket:     getEnv("OSS_BUCKET", "template-mall"),
		OSSUploadTTL:  getDuration("OSS_UPLOAD_TTL", 15*time.Minute),
		PublicBaseURL: trimSlash(getEnv("PUBLIC_BASE_URL", "http://127.0.0.1:8081")),
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

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
