package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	GRPCAddr string

	MySQLDSN string

	KafkaBrokers       []string
	KafkaTopic         string
	KafkaConsumerGroup string

	PayBaseURL     string
	PayHTTPTimeout time.Duration

	DownloadURLTTL time.Duration

	StorageProvider string
	OSSEndpoint     string
	OSSAccessKey    string
	OSSSecretKey    string
	OSSBucket       string
	OSSRegion       string
	// Mock 模式下下载签名密钥
	MockSignSecret string
}

func Load() Config {
	loadDotEnv()
	return Config{
		GRPCAddr: getEnv("GRPC_ADDR", ":50051"),

		MySQLDSN: getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/template_order_db?charset=utf8mb4&parseTime=True&loc=Local"),

		KafkaBrokers:       splitCSV(getEnv("KAFKA_BROKERS", "127.0.0.1:9092")),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "template-pay-events"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "template-order-consumer-group"),

		PayBaseURL:     getEnv("PAY_BASE_URL", "http://127.0.0.1:8082"),
		PayHTTPTimeout: getDurationEnv("PAY_HTTP_TIMEOUT", 5*time.Second),

		DownloadURLTTL: getDurationEnv("DOWNLOAD_URL_TTL", 30*time.Second),

		StorageProvider: getEnv("STORAGE_PROVIDER", "mock"),
		OSSEndpoint:     getEnv("OSS_ENDPOINT", ""),
		OSSAccessKey:    getEnv("OSS_ACCESS_KEY", ""),
		OSSSecretKey:    getEnv("OSS_SECRET_KEY", ""),
		OSSBucket:       getEnv("OSS_BUCKET", ""),
		OSSRegion:       getEnv("OSS_REGION", ""),
		MockSignSecret:  getEnv("MOCK_SIGN_SECRET", "dev-mock-sign-secret"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if sec, err := strconv.Atoi(v); err == nil {
		return time.Duration(sec) * time.Second
	}
	return def
}

func splitCSV(s string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			p := trimSpace(s[start:i])
			if p != "" {
				parts = append(parts, p)
			}
			start = i + 1
		}
	}
	return parts
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
