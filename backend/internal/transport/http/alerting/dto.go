package alerting

import appalerting "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/alerting"

// UpdateConfigRequest 是更新告警配置的请求体。指针字段表示是否更新该项。
type UpdateConfigRequest struct {
	Enabled          *bool     `json:"enabled"`
	EnabledNotifiers *[]string `json:"enabledNotifiers"`
	TelegramBotToken *string   `json:"telegramBotToken"`
	TelegramChatID   *string   `json:"telegramChatId"`
	WebhookURL       *string   `json:"webhookUrl"`
	DebounceSeconds  *int      `json:"debounceSeconds"`
}

func (r UpdateConfigRequest) toInput() appalerting.UpdateConfigInput {
	return appalerting.UpdateConfigInput{
		Enabled:          r.Enabled,
		EnabledNotifiers: r.EnabledNotifiers,
		TelegramBotToken: r.TelegramBotToken,
		TelegramChatID:   r.TelegramChatID,
		WebhookURL:       r.WebhookURL,
		DebounceSeconds:  r.DebounceSeconds,
	}
}

// TestTelegramRequest 是测试 Telegram 的可选覆盖参数。
type TestTelegramRequest struct {
	BotToken string `json:"telegramBotToken"`
	ChatID   string `json:"telegramChatId"`
}

// TestWebhookRequest 是测试 Webhook 的可选覆盖参数。
type TestWebhookRequest struct {
	URL string `json:"webhookUrl"`
}
