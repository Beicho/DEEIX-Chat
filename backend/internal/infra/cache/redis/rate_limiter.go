package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// rateLimiter 提供基于 Redis 的 HTTP 限流存储能力。
type rateLimiter struct {
	client *redis.Client
}

// NewRateLimiter 创建 Redis 限流器。
func NewRateLimiter(client *redis.Client) *rateLimiter {
	if client == nil {
		return nil
	}
	return &rateLimiter{client: client}
}

func userRateLimitOverrideKey(userID uint) string {
	return fmt.Sprintf("ratelimit:override:user:%d", userID)
}

// AllowSlidingWindow 使用有序集合实现滑动窗口限流。
func (r *rateLimiter) AllowSlidingWindow(ctx context.Context, key string, limit int, window time.Duration, ttl time.Duration) (bool, error) {
	if r == nil || r.client == nil || key == "" || limit <= 0 {
		return true, nil
	}
	if window <= 0 {
		window = time.Minute
	}
	if ttl <= 0 {
		ttl = window * 2
	}

	nowNanos := time.Now().UnixNano()
	now := nowNanos / int64(time.Millisecond)
	windowStart := now - window.Milliseconds()
	member := strconv.FormatInt(nowNanos, 10)

	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: member})
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, err
	}
	return countCmd.Val() < int64(limit), nil
}

// AllowFixedWindow 使用计数器实现固定窗口限流。
func (r *rateLimiter) AllowFixedWindow(ctx context.Context, keys []string, limit int, ttl time.Duration) (bool, error) {
	if r == nil || r.client == nil || len(keys) == 0 || limit <= 0 {
		return true, nil
	}
	if ttl <= 0 {
		ttl = time.Minute
	}

	pipe := r.client.Pipeline()
	incrCmds := make([]*redis.IntCmd, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		incrCmds = append(incrCmds, pipe.Incr(ctx, key))
		pipe.Expire(ctx, key, ttl)
	}
	if len(incrCmds) == 0 {
		return true, nil
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return true, err
	}

	for _, cmd := range incrCmds {
		if cmd.Val() > int64(limit) {
			return false, nil
		}
	}
	return true, nil
}

// SetUserRateLimitOverride stores a temporary per-user RPM override.
func (r *rateLimiter) SetUserRateLimitOverride(ctx context.Context, userID uint, rpm int, ttl time.Duration) error {
	if r == nil || r.client == nil || userID == 0 || rpm <= 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return r.client.Set(ctx, userRateLimitOverrideKey(userID), strconv.Itoa(rpm), ttl).Err()
}

// GetUserRateLimitOverride returns a temporary per-user RPM override when one exists.
func (r *rateLimiter) GetUserRateLimitOverride(ctx context.Context, userID uint) (int, bool, error) {
	if r == nil || r.client == nil || userID == 0 {
		return 0, false, nil
	}
	raw, err := r.client.Get(ctx, userRateLimitOverrideKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, err
	}
	rpm, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || rpm <= 0 {
		return 0, false, nil
	}
	return rpm, true, nil
}

// ClearUserRateLimitOverride removes a temporary per-user RPM override.
func (r *rateLimiter) ClearUserRateLimitOverride(ctx context.Context, userID uint) error {
	if r == nil || r.client == nil || userID == 0 {
		return nil
	}
	return r.client.Del(ctx, userRateLimitOverrideKey(userID)).Err()
}

// AcquireConcurrencySlot 占用一个并发槽位；超出上限时返回 false 且不占用。
func (r *rateLimiter) AcquireConcurrencySlot(ctx context.Context, key string, limit int, ttl time.Duration) (bool, error) {
	if r == nil || r.client == nil || key == "" || limit <= 0 {
		return true, nil
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	// 每次占用都续期，保证长任务期间槽位不过期；异常退出时由 TTL 兜底释放。
	if expireErr := r.client.Expire(ctx, key, ttl).Err(); expireErr != nil {
		return true, expireErr
	}
	if count > int64(limit) {
		r.client.Decr(ctx, key)
		return false, nil
	}
	return true, nil
}

// ReleaseConcurrencySlot 释放一个并发槽位。
func (r *rateLimiter) ReleaseConcurrencySlot(ctx context.Context, key string) error {
	if r == nil || r.client == nil || key == "" {
		return nil
	}
	count, err := r.client.Decr(ctx, key).Result()
	if err != nil {
		return err
	}
	if count <= 0 {
		return r.client.Del(ctx, key).Err()
	}
	return nil
}
