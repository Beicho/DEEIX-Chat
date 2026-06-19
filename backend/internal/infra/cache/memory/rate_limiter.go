package memory

import (
	"context"
	"strings"
	"time"
)

type fixedWindowCounter struct {
	count     int
	expiresAt time.Time
}

func (c *Cache) AllowSlidingWindow(ctx context.Context, key string, limit int, window time.Duration, ttl time.Duration) (bool, error) {
	if c == nil || strings.TrimSpace(key) == "" || limit <= 0 {
		return true, nil
	}
	if window <= 0 {
		window = time.Minute
	}
	now := time.Now()
	cutoff := now.Add(-window)
	c.mu.Lock()
	defer c.mu.Unlock()
	events := c.slidingHTTP[key][:0]
	for _, item := range c.slidingHTTP[key] {
		if item.After(cutoff) {
			events = append(events, item)
		}
	}
	allowed := len(events) < limit
	if allowed {
		events = append(events, now)
	}
	c.slidingHTTP[key] = events
	c.maybeSweepLocked(now)
	return allowed, nil
}

func (c *Cache) AllowFixedWindow(ctx context.Context, keys []string, limit int, ttl time.Duration) (bool, error) {
	if c == nil || len(keys) == 0 || limit <= 0 {
		return true, nil
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	allowed := true
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		item := c.fixedHTTP[key]
		if now.After(item.expiresAt) {
			item = fixedWindowCounter{expiresAt: now.Add(ttl)}
		}
		item.count++
		if item.count > limit {
			allowed = false
		}
		c.fixedHTTP[key] = item
	}
	c.maybeSweepLocked(now)
	return allowed, nil
}

func (c *Cache) SetUserRateLimitOverride(ctx context.Context, userID uint, rpm int, ttl time.Duration) error {
	if c == nil || userID == 0 || rpm <= 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rateLimitOverrides[userID] = rateLimitOverride{rpm: rpm, expiresAt: now.Add(ttl)}
	c.maybeSweepLocked(now)
	return nil
}

func (c *Cache) GetUserRateLimitOverride(ctx context.Context, userID uint) (int, bool, error) {
	if c == nil || userID == 0 {
		return 0, false, nil
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.rateLimitOverrides[userID]
	if !ok {
		return 0, false, nil
	}
	if now.After(item.expiresAt) {
		delete(c.rateLimitOverrides, userID)
		return 0, false, nil
	}
	c.maybeSweepLocked(now)
	return item.rpm, item.rpm > 0, nil
}

func (c *Cache) ClearUserRateLimitOverride(ctx context.Context, userID uint) error {
	if c == nil || userID == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rateLimitOverrides, userID)
	c.maybeSweepLocked(time.Now())
	return nil
}
