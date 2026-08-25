package alerting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
	"go.uber.org/zap"
)

// Dispatch 根据当前配置异步分发一次告警事件（含去抖）。
// 该方法立即返回，实际发送在后台 goroutine 中完成，不阻塞调用方（熔断热路径）。
func (s *Service) Dispatch(event AlertEvent) {
	if s == nil || s.store == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.dispatchSync(ctx, event)
	}()
}

// dispatchSync 同步执行分发逻辑（供 Dispatch 与测试使用）。
func (s *Service) dispatchSync(ctx context.Context, event AlertEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if strings.TrimSpace(event.Message) == "" {
		event.Message = defaultAlertText(event)
	}

	cfg, err := s.LoadConfig(ctx)
	if err != nil {
		s.warn("alerting_load_config_failed", zap.Error(err))
		return
	}
	if !cfg.Enabled {
		return
	}

	notifiers := s.buildNotifiers(cfg)
	if len(notifiers) == 0 {
		return
	}

	if !s.debouncer.Allow(ctx, dedupeKey(event), cfg.DebounceWindow) {
		return
	}

	for _, notifier := range notifiers {
		sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		if err := notifier.Notify(sendCtx, event); err != nil {
			// 不记录敏感配置，仅记录通道类型与事件类型。
			s.warn("alerting_notify_failed",
				zap.String("notifier", notifier.Kind()),
				zap.String("event_type", event.Type),
				zap.Int("channel_id", event.ChannelID),
				zap.Error(err),
			)
		}
		cancel()
	}
}

// buildNotifiers 根据配置构造已启用且配置完整的通道。
func (s *Service) buildNotifiers(cfg Config) []Notifier {
	env, ssrf := s.outboundEnv()
	notifiers := make([]Notifier, 0, 2)
	if cfg.notifierEnabled(NotifierTelegram) && cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		notifiers = append(notifiers, TelegramNotifier{
			BotToken:    cfg.TelegramBotToken,
			ChatID:      cfg.TelegramChatID,
			Env:         env,
			SSRFEnabled: ssrf,
		})
	}
	if cfg.notifierEnabled(NotifierWebhook) && cfg.WebhookURL != "" {
		notifiers = append(notifiers, WebhookNotifier{
			URL:         cfg.WebhookURL,
			Env:         env,
			SSRFEnabled: ssrf,
		})
	}
	return notifiers
}

// SendTest 使用当前配置（必要时叠加临时覆盖）发送一条测试告警，绕过去抖。
func (s *Service) SendTest(ctx context.Context, kind string, override TestOverride) error {
	cfg, err := s.LoadConfig(ctx)
	if err != nil {
		return err
	}
	env, ssrf := s.outboundEnv()
	event := testEvent()

	switch kind {
	case NotifierTelegram:
		token := firstNonEmpty(override.TelegramBotToken, cfg.TelegramBotToken)
		chatID := firstNonEmpty(override.TelegramChatID, cfg.TelegramChatID)
		if token == "" || chatID == "" {
			return fmt.Errorf("%w: telegram bot token and chat id are required", ErrInvalidConfig)
		}
		return TelegramNotifier{BotToken: token, ChatID: chatID, Env: env, SSRFEnabled: ssrf}.Notify(ctx, event)
	case NotifierWebhook:
		url := firstNonEmpty(override.WebhookURL, cfg.WebhookURL)
		if url == "" {
			return fmt.Errorf("%w: webhook url is required", ErrInvalidConfig)
		}
		if err := validateWebhookURL(url, ssrf); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
		return WebhookNotifier{URL: url, Env: env, SSRFEnabled: ssrf}.Notify(ctx, event)
	default:
		return fmt.Errorf("%w: unknown notifier: %s", ErrInvalidConfig, kind)
	}
}

// TestOverride 允许在不落库的情况下临时覆盖凭据进行测试。
type TestOverride struct {
	TelegramBotToken string
	TelegramChatID   string
	WebhookURL       string
}

func testEvent() AlertEvent {
	return AlertEvent{
		Type:        EventTypeCircuitOpen,
		ChannelName: "测试渠道",
		ModelNames:  []string{"GPT-4", "Claude"},
		Timestamp:   time.Now(),
		Message:     "🔔 这是一条来自 DEEIX-Chat 的告警测试消息。",
	}
}

// defaultAlertText 在未显式提供 message 时构造默认文案。
func defaultAlertText(event AlertEvent) string {
	models := strings.Join(event.ModelNames, ", ")
	if strings.TrimSpace(models) == "" {
		models = "-"
	}
	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	timeStr := ts.Format("2006-01-02 15:04:05")
	// 风控类告警的文案由调用方直接给出，这里只补时间。
	if event.Type == EventTypeRiskSpend || event.Type == EventTypeRiskFingerprint {
		return fmt.Sprintf("%s\n时间: %s", firstNonEmpty(event.Message, "风控告警"), timeStr)
	}
	if event.Type == EventTypeCircuitClosed {
		return fmt.Sprintf("🟢 渠道恢复\n渠道: %s\n影响模型: %s\n时间: %s",
			displayName(event.ChannelName), models, timeStr)
	}
	return fmt.Sprintf("🔴 渠道熔断告警\n渠道: %s\n影响模型: %s\n时间: %s",
		displayName(event.ChannelName), models, timeStr)
}

func displayName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "-"
	}
	return name
}

// dedupeKey 构造去抖 key：按渠道 + 事件类型，使熔断/恢复各自独立去抖。
// 风控告警按事件类型 + 主体 ID（ChannelID 复用为 userID）去抖。
func dedupeKey(event AlertEvent) string {
	return fmt.Sprintf("%d:%s", event.ChannelID, event.Type)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func validateWebhookURL(raw string, ssrfEnabled bool) error {
	if err := security.ValidateOutboundHTTPURL(raw, security.NewStrictOutboundPolicy(ssrfEnabled)); err != nil {
		return fmt.Errorf("webhook url is not allowed")
	}
	return nil
}
