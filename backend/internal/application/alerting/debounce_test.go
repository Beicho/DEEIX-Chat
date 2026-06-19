package alerting

import (
	"context"
	"testing"
	"time"
)

func TestMemoryDebouncerAllowsOncePerWindow(t *testing.T) {
	d := NewMemoryDebouncer()
	ctx := context.Background()
	key := "1:circuit_open"

	if !d.Allow(ctx, key, time.Minute) {
		t.Fatal("first call should be allowed")
	}
	if d.Allow(ctx, key, time.Minute) {
		t.Fatal("second call within window should be denied")
	}
}

func TestMemoryDebouncerDistinctKeysIndependent(t *testing.T) {
	d := NewMemoryDebouncer()
	ctx := context.Background()

	if !d.Allow(ctx, "1:circuit_open", time.Minute) {
		t.Fatal("open key first call should be allowed")
	}
	if !d.Allow(ctx, "1:circuit_closed", time.Minute) {
		t.Fatal("closed key should not be affected by open key debounce")
	}
	if !d.Allow(ctx, "2:circuit_open", time.Minute) {
		t.Fatal("different channel should debounce independently")
	}
}

func TestMemoryDebouncerReallowsAfterWindow(t *testing.T) {
	d := NewMemoryDebouncer()
	ctx := context.Background()
	key := "1:circuit_open"

	if !d.Allow(ctx, key, 20*time.Millisecond) {
		t.Fatal("first call should be allowed")
	}
	if d.Allow(ctx, key, 20*time.Millisecond) {
		t.Fatal("second call within window should be denied")
	}
	time.Sleep(40 * time.Millisecond)
	if !d.Allow(ctx, key, 20*time.Millisecond) {
		t.Fatal("call after window elapsed should be allowed again")
	}
}

func TestRedisDebouncerFallsBackToMemoryWhenClientNil(t *testing.T) {
	d := NewRedisDebouncer(nil)
	ctx := context.Background()
	key := "9:circuit_open"

	if !d.Allow(ctx, key, time.Minute) {
		t.Fatal("first call should be allowed via memory fallback")
	}
	if d.Allow(ctx, key, time.Minute) {
		t.Fatal("second call should be denied via memory fallback")
	}
}
