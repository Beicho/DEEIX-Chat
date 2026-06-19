package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

// telegramAPIBase 是 Telegram Bot API 的基础地址。
const telegramAPIBase = "https://api.telegram.org"

// TelegramNotifier 通过 Telegram Bot API 发送告警。
type TelegramNotifier struct {
	BotToken    string
	ChatID      string
	Env         string
	SSRFEnabled bool
}

// Kind 返回通道标识。
func (n TelegramNotifier) Kind() string { return NotifierTelegram }

// Notify 发送 Telegram 告警消息。
func (n TelegramNotifier) Notify(ctx context.Context, event AlertEvent) error {
	token := strings.TrimSpace(n.BotToken)
	chatID := strings.TrimSpace(n.ChatID)
	if token == "" || chatID == "" {
		return fmt.Errorf("telegram notifier is not configured")
	}

	payload := map[string]string{
		"chat_id": chatID,
		"text":    formatTelegramMessage(event),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := telegramAPIBase + "/bot" + token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := security.NewOutboundHTTPClient(n.Env, n.SSRFEnabled, 10*time.Second)
	resp, err := client.Do(req)
	if err != nil {
		// 不回显 token：错误里 endpoint 含 token，需脱敏。
		return fmt.Errorf("telegram request failed: %s", redactToken(err.Error(), token))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := readErrorSnippet(resp.Body)
		return fmt.Errorf("telegram api returned status %d: %s", resp.StatusCode, snippet)
	}
	return nil
}

// formatTelegramMessage 构造人类可读的 Telegram 文案。
func formatTelegramMessage(event AlertEvent) string {
	if strings.TrimSpace(event.Message) != "" {
		return event.Message
	}
	return defaultAlertText(event)
}

// redactToken 将文本中出现的 token 替换为脱敏占位符。
func redactToken(text string, token string) string {
	if strings.TrimSpace(token) == "" {
		return text
	}
	return strings.ReplaceAll(text, token, "***")
}

// readErrorSnippet 读取上游错误响应的前若干字节用于日志/错误（限长，避免泄露过多）。
func readErrorSnippet(r io.Reader) string {
	const maxBytes = 256
	buf, _ := io.ReadAll(io.LimitReader(r, maxBytes))
	return strings.TrimSpace(string(buf))
}
