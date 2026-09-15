package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr string

	MySQLDSN string

	KafkaBrokers []string
	KafkaTopic   string

	PublicBaseURL   string
	MallFrontendURL string
	KafkaTimeout    time.Duration
}

func Load() Config {
	loadDotEnv()
	return Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8082"),
		MySQLDSN:        getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/template_order_db?charset=utf8mb4&parseTime=True&loc=Local"),
		KafkaBrokers:    splitCSV(getEnv("KAFKA_BROKERS", "127.0.0.1:9092")),
		KafkaTopic:      getEnv("KAFKA_TOPIC", "template-pay-events"),
		PublicBaseURL:   strings.TrimRight(getEnv("PUBLIC_BASE_URL", "http://127.0.0.1:8082"), "/"),
		MallFrontendURL: strings.TrimRight(getEnv("MALL_FRONTEND_URL", "http://127.0.0.1:3000"), "/"),
		KafkaTimeout:    getDurationEnv("KAFKA_TIMEOUT", 5*time.Second),
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
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
