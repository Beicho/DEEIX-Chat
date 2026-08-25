package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

// WebhookNotifier 向配置的 URL POST JSON 告警载荷。
type WebhookNotifier struct {
	URL         string
	Env         string
	SSRFEnabled bool
}

// Kind 返回通道标识。
func (n WebhookNotifier) Kind() string { return NotifierWebhook }

// webhookPayload 是 webhook 通道的 JSON 载荷结构（不含任何内部路由/上游商信息）。
type webhookPayload struct {
	Type        string   `json:"type"`
	ChannelName string   `json:"channel_name"`
	Models      []string `json:"models"`
	Timestamp   string   `json:"timestamp"`
	Message     string   `json:"message"`
}

// Notify 向 webhook 地址发送告警。
func (n WebhookNotifier) Notify(ctx context.Context, event AlertEvent) error {
	target := strings.TrimSpace(n.URL)
	if target == "" {
		return fmt.Errorf("webhook notifier is not configured")
	}
	policy := security.NewStrictOutboundPolicy(n.SSRFEnabled)
	if err := security.ValidateOutboundHTTPURL(target, policy); err != nil {
		return fmt.Errorf("webhook url is not allowed")
	}

	models := event.ModelNames
	if models == nil {
		models = []string{}
	}
	message := strings.TrimSpace(event.Message)
	if message == "" {
		message = defaultAlertText(event)
	}
	payload := webhookPayload{
		Type:        event.Type,
		ChannelName: event.ChannelName,
		Models:      models,
		Timestamp:   event.Timestamp.UTC().Format(time.RFC3339),
		Message:     message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := security.NewOutboundHTTPClient(policy, 10*time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
