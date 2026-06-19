package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
)

func TestIsNewDeviceLoginRequiresExistingDifferentSession(t *testing.T) {
	now := time.Now()
	user := &domainuser.User{ID: 7, Status: domainuser.StatusActive}
	current := &domainuser.Session{
		SessionID:   "current",
		UserID:      7,
		ClientIP:    "198.51.100.10",
		UserAgent:   "Mozilla/5.0 Chrome/120",
		DeviceName:  "Chrome on macOS",
		BrowserName: "Chrome",
		OSName:      "macOS",
		DeviceType:  "desktop",
		ExpiresAt:   now.Add(time.Hour),
	}

	if isNewDeviceLogin(user, current, nil) {
		t.Fatal("first login should not be treated as a new device")
	}
	if isNewDeviceLogin(user, current, []domainuser.Session{*current}) {
		t.Fatal("same session signature should not be treated as a new device")
	}

	existing := domainuser.Session{
		SessionID:   "existing",
		UserID:      7,
		ClientIP:    "203.0.113.5",
		UserAgent:   "Mozilla/5.0 Safari/17",
		DeviceName:  "Safari on iOS",
		BrowserName: "Safari",
		OSName:      "iOS",
		DeviceType:  "mobile",
		ExpiresAt:   now.Add(time.Hour),
	}
	if !isNewDeviceLogin(user, current, []domainuser.Session{existing}) {
		t.Fatal("different active session signature should be treated as a new device")
	}
}

func TestBuildNewDeviceNotificationBodyIncludesDeviceLocationAndIP(t *testing.T) {
	body := buildNewDeviceNotificationBody(sessionAuditSnapshot{
		DeviceName:  "Chrome on macOS",
		ClientIP:    "198.51.100.10",
		CountryCode: "US",
		RegionName:  "California",
		CityName:    "San Francisco",
	})

	for _, expected := range []string{"Chrome on macOS", "San Francisco, California, US", "198.51.100.10"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body %q missing %q", body, expected)
		}
	}
}

func TestNewDeviceNotificationSourceIDDoesNotExposeSessionID(t *testing.T) {
	now := time.Date(2026, 6, 15, 8, 0, 0, 123, time.UTC)
	service := &Service{}
	notifier := &fakeAuthNotificationNotifier{}
	service.SetNotificationNotifier(notifier)
	current := &domainuser.Session{
		SessionID:   "current-session-token",
		UserID:      7,
		ClientIP:    "198.51.100.10",
		UserAgent:   "Mozilla/5.0 Chrome/120",
		DeviceName:  "Chrome on macOS",
		BrowserName: "Chrome",
		OSName:      "macOS",
		DeviceType:  "desktop",
		ExpiresAt:   now.Add(time.Hour),
	}
	existing := domainuser.Session{
		SessionID:   "existing",
		UserID:      7,
		ClientIP:    "203.0.113.5",
		UserAgent:   "Mozilla/5.0 Safari/17",
		DeviceName:  "Safari on iOS",
		BrowserName: "Safari",
		OSName:      "iOS",
		DeviceType:  "mobile",
		ExpiresAt:   now.Add(time.Hour),
	}

	service.notifyNewDeviceLogin(context.Background(), &domainuser.User{ID: 7}, current, []domainuser.Session{existing}, sessionAuditSnapshot{}, now)

	if notifier.input.SourceID == "" {
		t.Fatal("source ID is empty")
	}
	if strings.Contains(notifier.input.SourceID, current.SessionID) {
		t.Fatalf("source ID exposes session ID: %q", notifier.input.SourceID)
	}
	if !strings.HasPrefix(notifier.input.SourceID, "new_device:") {
		t.Fatalf("source ID = %q, want new_device prefix", notifier.input.SourceID)
	}
}

type fakeAuthNotificationNotifier struct {
	input appnotification.SystemNotificationInput
}

func (n *fakeAuthNotificationNotifier) CreateSystemNotification(_ context.Context, _ uint, input appnotification.SystemNotificationInput) (*appnotification.NotificationView, error) {
	n.input = input
	return &appnotification.NotificationView{}, nil
}
