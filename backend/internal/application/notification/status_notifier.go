package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

// StatusNotifier is the extension point for future status alerts.
type StatusNotifier interface {
	Notify(ctx context.Context, title string, message string) error
}

type CompositeStatusNotifier struct {
	items []StatusNotifier
}

func NewStatusNotifier(cfg config.Config) StatusNotifier {
	items := make([]StatusNotifier, 0, 2)
	if strings.TrimSpace(cfg.StatusNotifierWebhookURL) != "" {
		items = append(items, WebhookStatusNotifier{
			URL:        strings.TrimSpace(cfg.StatusNotifierWebhookURL),
			HTTPClient: security.NewOutboundHTTPClient(cfg.Env, cfg.SSRFProtectionEnabled, 10*time.Second),
		})
	}
	if strings.TrimSpace(cfg.StatusNotifierEmail) != "" {
		items = append(items, EmailStatusNotifier{
			Recipient: strings.TrimSpace(cfg.StatusNotifierEmail),
			Host:      strings.TrimSpace(cfg.SMTPHost),
			Port:      cfg.SMTPPort,
			Username:  strings.TrimSpace(cfg.SMTPUsername),
			Password:  strings.TrimSpace(cfg.SMTPPassword),
			From:      strings.TrimSpace(cfg.SMTPFrom),
		})
	}
	return CompositeStatusNotifier{items: items}
}

func (n CompositeStatusNotifier) Notify(ctx context.Context, title string, message string) error {
	for _, item := range n.items {
		if err := item.Notify(ctx, title, message); err != nil {
			return err
		}
	}
	return nil
}

type WebhookStatusNotifier struct {
	URL        string
	HTTPClient *http.Client
}

func (n WebhookStatusNotifier) Notify(ctx context.Context, title string, message string) error {
	if strings.TrimSpace(n.URL) == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"title": title, "message": message})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := n.HTTPClient
	if client == nil {
		client = security.NewOutboundHTTPClient("", false, 10*time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type EmailStatusNotifier struct {
	Recipient string
	Host      string
	Port      int
	Username  string
	Password  string
	From      string
}

func (n EmailStatusNotifier) Notify(ctx context.Context, title string, message string) error {
	if strings.TrimSpace(n.Recipient) == "" || strings.TrimSpace(n.Host) == "" || strings.TrimSpace(n.From) == "" {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	port := n.Port
	if port <= 0 {
		port = 587
	}
	addr := n.Host + ":" + strconv.Itoa(port)
	var auth smtp.Auth
	if n.Username != "" || n.Password != "" {
		auth = smtp.PlainAuth("", n.Username, n.Password, n.Host)
	}
	body := []byte("To: " + n.Recipient + "\r\nSubject: " + title + "\r\n\r\n" + message)
	return smtp.SendMail(addr, auth, n.From, []string{n.Recipient}, body)
}
