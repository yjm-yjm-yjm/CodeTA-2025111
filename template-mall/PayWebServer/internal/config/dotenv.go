package config

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv 读取工作目录 .env；已存在的环境变量优先（便于 Docker/Compose 覆盖）。
func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		if k == "" {
			continue
		}
		if _, ok := os.LookupEnv(k); ok {
			continue
		}
		_ = os.Setenv(k, v)
	}
}
