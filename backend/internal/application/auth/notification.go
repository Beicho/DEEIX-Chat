package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
)

const (
	authNotificationSource       = "auth"
	authNewDeviceNotificationURL = "/setting/account"
	authNewDeviceTitle           = "新设备登录提醒"
	authNewDeviceBodyFallback    = "你的账号刚刚从新设备登录。若不是本人操作，请立即修改密码。"
)

type authNotificationNotifier interface {
	CreateSystemNotification(ctx context.Context, userID uint, input appnotification.SystemNotificationInput) (*appnotification.NotificationView, error)
}

func (s *Service) notifyNewDeviceLogin(
	ctx context.Context,
	user *domainuser.User,
	currentSession *domainuser.Session,
	existingSessions []domainuser.Session,
	snapshot sessionAuditSnapshot,
	now time.Time,
) {
	if s == nil || s.notificationNotifier == nil || user == nil || currentSession == nil {
		return
	}
	if !isNewDeviceLogin(user, currentSession, existingSessions) {
		return
	}
	input := appnotification.SystemNotificationInput{
		Type:      domainnotification.TypeAuth,
		Title:     authNewDeviceTitle,
		Body:      buildNewDeviceNotificationBody(snapshot),
		ActionURL: authNewDeviceNotificationURL,
		Source:    authNotificationSource,
		SourceID:  newDeviceNotificationSourceID(now),
		Metadata: map[string]any{
			"session_id":     strings.TrimSpace(currentSession.SessionID),
			"device_name":    snapshot.DeviceName,
			"browser_name":   snapshot.BrowserName,
			"os_name":        snapshot.OSName,
			"device_type":    snapshot.DeviceType,
			"client_ip":      snapshot.ClientIP,
			"location_label": requestLocationLabel(snapshot),
			"geo_source":     snapshot.GeoSource,
			"geo_accuracy":   snapshot.GeoAccuracy,
		},
	}
	_, _ = s.notificationNotifier.CreateSystemNotification(ctx, user.ID, input)
}

func newDeviceNotificationSourceID(now time.Time) string {
	return fmt.Sprintf("new_device:%d", now.UnixNano())
}

func isNewDeviceLogin(user *domainuser.User, currentSession *domainuser.Session, existingSessions []domainuser.Session) bool {
	if user == nil || currentSession == nil {
		return false
	}
	if len(existingSessions) == 0 {
		return false
	}
	currentSignature := sessionLoginSignature(*currentSession)
	comparableSessions := 0
	for _, session := range existingSessions {
		if strings.TrimSpace(session.SessionID) == strings.TrimSpace(currentSession.SessionID) {
			continue
		}
		comparableSessions++
		if sessionLoginSignature(session) == currentSignature {
			return false
		}
	}
	return comparableSessions > 0
}

func sessionLoginSignature(session domainuser.Session) string {
	return strings.Join([]string{
		strings.TrimSpace(session.ClientIP),
		strings.TrimSpace(session.UserAgent),
		strings.TrimSpace(session.DeviceName),
		strings.TrimSpace(session.BrowserName),
		strings.TrimSpace(session.OSName),
		strings.TrimSpace(session.DeviceType),
	}, "|")
}

func buildNewDeviceNotificationBody(snapshot sessionAuditSnapshot) string {
	parts := make([]string, 0, 3)
	if value := strings.TrimSpace(snapshot.DeviceName); value != "" {
		parts = append(parts, value)
	}
	if value := requestLocationLabel(snapshot); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(snapshot.ClientIP); value != "" {
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return authNewDeviceBodyFallback
	}
	return "你的账号刚刚从 " + strings.Join(parts, " / ") + " 登录。若不是本人操作，请立即修改密码。"
}

func requestLocationLabel(snapshot sessionAuditSnapshot) string {
	parts := make([]string, 0, 3)
	if city := strings.TrimSpace(snapshot.CityName); city != "" {
		parts = append(parts, city)
	}
	if region := strings.TrimSpace(snapshot.RegionName); region != "" {
		parts = append(parts, region)
	}
	if country := strings.TrimSpace(snapshot.CountryCode); country != "" {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}
