package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"go.uber.org/zap"
)

// 配置键（namespace=alerting）。
const (
	KeyEnabled          = "enabled"
	KeyEnabledNotifiers = "enabled_notifiers"
	KeyTelegramBotToken = "telegram_bot_token"
	KeyTelegramChatID   = "telegram_chat_id"
	KeyWebhookURL       = "webhook_url"
	KeyDebounceSeconds  = "debounce_seconds"
)

// SettingPatch 描述一次告警配置写入。
type SettingPatch struct {
	Key   string
	Value string
	Clear bool
}

// ConfigStore 抽象告警配置的读写（由 settings 模块适配实现）。
type ConfigStore interface {
	// Load 返回 alerting namespace 下的运行时配置（敏感项已解密）。
	Load(ctx context.Context) (map[string]string, error)
	// Save 写入 alerting namespace 配置补丁（敏感项由实现负责加密）。
	Save(ctx context.Context, patches []SettingPatch) error
}

// Config 是解析后的告警运行时配置。
type Config struct {
	Enabled          bool
	EnabledNotifiers []string
	TelegramBotToken string
	TelegramChatID   string
	WebhookURL       string
	DebounceWindow   time.Duration
}

// ConfigView 是返回给管理端的脱敏配置视图。
type ConfigView struct {
	Enabled            bool     `json:"enabled"`
	EnabledNotifiers   []string `json:"enabledNotifiers"`
	TelegramConfigured bool     `json:"telegramConfigured"`
	TelegramChatID     string   `json:"telegramChatId"`
	WebhookConfigured  bool     `json:"webhookConfigured"`
	DebounceSeconds    int      `json:"debounceSeconds"`
}

// Service 协调告警配置、去抖与多通道分发。
type Service struct {
	store     ConfigStore
	debouncer Debouncer
	cfg       *config.Runtime
	logger    *zap.Logger
}

// NewService 创建告警服务。
func NewService(store ConfigStore, debouncer Debouncer, cfg *config.Runtime, logger *zap.Logger) *Service {
	if debouncer == nil {
		debouncer = NewMemoryDebouncer()
	}
	return &Service{store: store, debouncer: debouncer, cfg: cfg, logger: logger}
}

func (s *Service) outboundEnv() (string, bool) {
	if s == nil || s.cfg == nil {
		return "", false
	}
	snapshot := s.cfg.Snapshot()
	return snapshot.Env, snapshot.SSRFProtectionEnabled
}

func (s *Service) warn(message string, fields ...zap.Field) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Warn(message, fields...)
}

// LoadConfig 读取并解析当前告警配置。
func (s *Service) LoadConfig(ctx context.Context) (Config, error) {
	values, err := s.store.Load(ctx)
	if err != nil {
		return Config{}, err
	}
	return parseConfig(values), nil
}

func parseConfig(values map[string]string) Config {
	enabled, _ := strconv.ParseBool(strings.TrimSpace(values[KeyEnabled]))
	debounceSeconds, _ := strconv.Atoi(strings.TrimSpace(values[KeyDebounceSeconds]))
	window := time.Duration(debounceSeconds) * time.Second
	if window <= 0 {
		window = defaultDebounceWindow
	}
	return Config{
		Enabled:          enabled,
		EnabledNotifiers: parseEnabledNotifiers(values[KeyEnabledNotifiers]),
		TelegramBotToken: strings.TrimSpace(values[KeyTelegramBotToken]),
		TelegramChatID:   strings.TrimSpace(values[KeyTelegramChatID]),
		WebhookURL:       strings.TrimSpace(values[KeyWebhookURL]),
		DebounceWindow:   window,
	}
}

// parseEnabledNotifiers 解析 JSON 数组或逗号分隔的通道列表。
func parseEnabledNotifiers(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return nil
		}
	} else {
		arr = strings.Split(raw, ",")
	}
	seen := make(map[string]struct{}, len(arr))
	results := make([]string, 0, len(arr))
	for _, item := range arr {
		v := strings.ToLower(strings.TrimSpace(item))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		results = append(results, v)
	}
	sort.Strings(results)
	return results
}

func (c Config) notifierEnabled(kind string) bool {
	for _, item := range c.EnabledNotifiers {
		if item == kind {
			return true
		}
	}
	return false
}

// GetConfigView 返回脱敏的配置视图。
func (s *Service) GetConfigView(ctx context.Context) (ConfigView, error) {
	cfg, err := s.LoadConfig(ctx)
	if err != nil {
		return ConfigView{}, err
	}
	return ConfigView{
		Enabled:            cfg.Enabled,
		EnabledNotifiers:   cfg.EnabledNotifiers,
		TelegramConfigured: cfg.TelegramBotToken != "" && cfg.TelegramChatID != "",
		TelegramChatID:     cfg.TelegramChatID,
		WebhookConfigured:  cfg.WebhookURL != "",
		DebounceSeconds:    int(cfg.DebounceWindow / time.Second),
	}, nil
}

// UpdateConfigInput 描述一次配置更新（指针字段表示是否更新）。
type UpdateConfigInput struct {
	Enabled          *bool
	EnabledNotifiers *[]string
	TelegramBotToken *string
	TelegramChatID   *string
	WebhookURL       *string
	DebounceSeconds  *int
}

// UpdateConfig 校验并持久化配置更新。
func (s *Service) UpdateConfig(ctx context.Context, input UpdateConfigInput) (ConfigView, error) {
	patches := make([]SettingPatch, 0, 6)

	if input.Enabled != nil {
		patches = append(patches, SettingPatch{Key: KeyEnabled, Value: strconv.FormatBool(*input.Enabled)})
	}
	if input.EnabledNotifiers != nil {
		normalized := parseEnabledNotifiers(strings.Join(*input.EnabledNotifiers, ","))
		for _, item := range normalized {
			if item != NotifierTelegram && item != NotifierWebhook {
				return ConfigView{}, fmt.Errorf("%w: enabled_notifiers must contain only: telegram, webhook", ErrInvalidConfig)
			}
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return ConfigView{}, err
		}
		patches = append(patches, SettingPatch{Key: KeyEnabledNotifiers, Value: string(encoded)})
	}
	if input.TelegramBotToken != nil {
		value := strings.TrimSpace(*input.TelegramBotToken)
		patches = append(patches, SettingPatch{Key: KeyTelegramBotToken, Value: value, Clear: value == ""})
	}
	if input.TelegramChatID != nil {
		patches = append(patches, SettingPatch{Key: KeyTelegramChatID, Value: strings.TrimSpace(*input.TelegramChatID)})
	}
	if input.WebhookURL != nil {
		value := strings.TrimSpace(*input.WebhookURL)
		if value != "" {
			env, ssrf := s.outboundEnv()
			if err := validateWebhookURL(value, env, ssrf); err != nil {
				return ConfigView{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
			}
		}
		patches = append(patches, SettingPatch{Key: KeyWebhookURL, Value: value, Clear: value == ""})
	}
	if input.DebounceSeconds != nil {
		seconds := *input.DebounceSeconds
		if seconds < 30 || seconds > 3600 {
			return ConfigView{}, fmt.Errorf("%w: debounce_seconds must be between 30 and 3600", ErrInvalidConfig)
		}
		patches = append(patches, SettingPatch{Key: KeyDebounceSeconds, Value: strconv.Itoa(seconds)})
	}

	if len(patches) > 0 {
		if err := s.store.Save(ctx, patches); err != nil {
			return ConfigView{}, err
		}
	}
	return s.GetConfigView(ctx)
}
