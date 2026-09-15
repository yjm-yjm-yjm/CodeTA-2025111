package idgen

import (
	"fmt"
	"sync/atomic"
	"time"
)

var seq uint64

// NewBusinessID 生成业务编号：prefix + 时间 + 单调序号。
func NewBusinessID(prefix string) string {
	n := atomic.AddUint64(&seq, 1)
	return fmt.Sprintf("%s%s%06d", prefix, time.Now().Format("20060102150405"), n%1_000_000)
}
