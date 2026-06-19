package alerting

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Debouncer 决定某个去抖 key 当前是否允许发送告警。
type Debouncer interface {
	// Allow 在 window 内对同一 key 仅返回一次 true。
	Allow(ctx context.Context, key string, window time.Duration) bool
}

// redisDebouncer 使用 Redis SET NX EX 实现跨实例去抖。
type redisDebouncer struct {
	client   *redis.Client
	fallback *memoryDebouncer
}

// NewRedisDebouncer 创建基于 Redis 的去抖器；client 为 nil 时退化为内存实现。
func NewRedisDebouncer(client *redis.Client) Debouncer {
	return &redisDebouncer{client: client, fallback: newMemoryDebouncer()}
}

// NewMemoryDebouncer 创建纯内存去抖器（单实例 / Redis 不可用时使用）。
func NewMemoryDebouncer() Debouncer {
	return newMemoryDebouncer()
}

func (d *redisDebouncer) Allow(ctx context.Context, key string, window time.Duration) bool {
	if window <= 0 {
		window = defaultDebounceWindow
	}
	if d.client == nil {
		return d.fallback.Allow(ctx, key, window)
	}
	ok, err := d.client.SetNX(ctx, alertDebounceKey(key), "1", window).Result()
	if err != nil {
		// Redis 异常时退化为内存去抖，保证不丢告警的同时仍有基本节流。
		return d.fallback.Allow(ctx, key, window)
	}
	return ok
}

// memoryDebouncer 是单实例内存去抖实现。
type memoryDebouncer struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func newMemoryDebouncer() *memoryDebouncer {
	return &memoryDebouncer{seen: make(map[string]time.Time)}
}

func (d *memoryDebouncer) Allow(ctx context.Context, key string, window time.Duration) bool {
	if window <= 0 {
		window = defaultDebounceWindow
	}
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sweepLocked(now)
	if last, ok := d.seen[key]; ok && now.Sub(last) < window {
		return false
	}
	d.seen[key] = now
	return true
}

// sweepLocked 清理过期条目，避免内存无限增长。
func (d *memoryDebouncer) sweepLocked(now time.Time) {
	for key, ts := range d.seen {
		if now.Sub(ts) > maxDebounceRetention {
			delete(d.seen, key)
		}
	}
}

const (
	defaultDebounceWindow = 5 * time.Minute
	maxDebounceRetention  = time.Hour
)

func alertDebounceKey(key string) string {
	return "alert:debounce:" + key
}
