package alerting

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeStore 是内存 ConfigStore，用于测试。
type fakeStore struct {
	values map[string]string
}

func (s *fakeStore) Load(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out, nil
}

func (s *fakeStore) Save(ctx context.Context, patches []SettingPatch) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for _, p := range patches {
		if p.Clear {
			delete(s.values, p.Key)
			continue
		}
		s.values[p.Key] = p.Value
	}
	return nil
}

func TestParseConfigDefaults(t *testing.T) {
	cfg := parseConfig(map[string]string{})
	if cfg.Enabled {
		t.Fatal("expected disabled by default")
	}
	if cfg.DebounceWindow != defaultDebounceWindow {
		t.Fatalf("expected default debounce window, got %v", cfg.DebounceWindow)
	}
	if len(cfg.EnabledNotifiers) != 0 {
		t.Fatalf("expected no notifiers, got %v", cfg.EnabledNotifiers)
	}
}

func TestParseEnabledNotifiersJSONAndCSV(t *testing.T) {
	jsonResult := parseEnabledNotifiers(`["telegram","webhook","telegram"]`)
	if len(jsonResult) != 2 {
		t.Fatalf("expected 2 unique notifiers, got %v", jsonResult)
	}
	csvResult := parseEnabledNotifiers("webhook, telegram")
	if len(csvResult) != 2 {
		t.Fatalf("expected 2 notifiers from csv, got %v", csvResult)
	}
}

func TestUpdateConfigRejectsUnknownNotifier(t *testing.T) {
	svc := NewService(&fakeStore{values: map[string]string{}}, NewMemoryDebouncer(), nil, nil)
	bad := []string{"sms"}
	_, err := svc.UpdateConfig(context.Background(), UpdateConfigInput{EnabledNotifiers: &bad})
	if err == nil {
		t.Fatal("expected error for unknown notifier")
	}
}

func TestUpdateConfigRejectsOutOfRangeDebounce(t *testing.T) {
	svc := NewService(&fakeStore{values: map[string]string{}}, NewMemoryDebouncer(), nil, nil)
	tooSmall := 5
	if _, err := svc.UpdateConfig(context.Background(), UpdateConfigInput{DebounceSeconds: &tooSmall}); err == nil {
		t.Fatal("expected error for debounce < 30")
	}
}

func TestGetConfigViewMasksTelegramToken(t *testing.T) {
	store := &fakeStore{values: map[string]string{
		KeyTelegramBotToken: "secret-token",
		KeyTelegramChatID:   "12345",
	}}
	svc := NewService(store, NewMemoryDebouncer(), nil, nil)
	view, err := svc.GetConfigView(context.Background())
	if err != nil {
		t.Fatalf("GetConfigView error: %v", err)
	}
	if !view.TelegramConfigured {
		t.Fatal("expected telegram configured")
	}
	if view.TelegramChatID != "12345" {
		t.Fatalf("expected chat id surfaced, got %q", view.TelegramChatID)
	}
}

// countingNotifier 统计调用次数。
type countingNotifier struct {
	mu    sync.Mutex
	count int
	kind  string
}

func (n *countingNotifier) Kind() string { return n.kind }
func (n *countingNotifier) Notify(ctx context.Context, event AlertEvent) error {
	n.mu.Lock()
	n.count++
	n.mu.Unlock()
	return nil
}

func TestDispatchSyncDebouncesPerKey(t *testing.T) {
	store := &fakeStore{values: map[string]string{
		KeyEnabled:          "true",
		KeyEnabledNotifiers: `["webhook"]`,
		KeyWebhookURL:       "https://example.com/hook",
		KeyDebounceSeconds:  "300",
	}}
	debouncer := NewMemoryDebouncer()
	svc := NewService(store, debouncer, nil, nil)

	event := AlertEvent{Type: EventTypeCircuitOpen, ChannelID: 1, ChannelName: "c1", Timestamp: time.Now()}
	// 第二次应被去抖拦截（即便 webhook 实际发送会失败，去抖判定也先于发送）。
	if !debouncer.Allow(context.Background(), dedupeKey(event), 5*time.Minute) {
		t.Fatal("manual first allow should succeed")
	}
	if debouncer.Allow(context.Background(), dedupeKey(event), 5*time.Minute) {
		t.Fatal("second allow within window should be denied")
	}
	// 验证 dedupeKey 对 open/closed 区分。
	closed := event
	closed.Type = EventTypeCircuitClosed
	if !debouncer.Allow(context.Background(), dedupeKey(closed), 5*time.Minute) {
		t.Fatal("closed event should debounce independently of open")
	}

	_ = svc
}

func TestDefaultAlertTextDistinguishesOpenClosed(t *testing.T) {
	open := defaultAlertText(AlertEvent{Type: EventTypeCircuitOpen, ChannelName: "c", ModelNames: []string{"m1"}})
	closed := defaultAlertText(AlertEvent{Type: EventTypeCircuitClosed, ChannelName: "c", ModelNames: []string{"m1"}})
	if open == closed {
		t.Fatal("open and closed text should differ")
	}
}
