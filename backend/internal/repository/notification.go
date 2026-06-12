package repository

import (
	"context"
	"time"

	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
)

// NotificationRepository 定义站内通知持久化能力。
type NotificationRepository interface {
	CreateNotification(ctx context.Context, item *domainnotification.Notification) (*domainnotification.Notification, error)
	ListNotifications(ctx context.Context, userID uint, filter NotificationListFilter, offset int, limit int) ([]domainnotification.Notification, int64, error)
	CountUnreadNotifications(ctx context.Context, userID uint) (int64, error)
	MarkNotificationRead(ctx context.Context, userID uint, notificationID uint, now time.Time) error
	MarkAllNotificationsRead(ctx context.Context, userID uint, now time.Time) error
}

// NotificationListFilter 描述通知列表筛选条件。
type NotificationListFilter struct {
	UnreadOnly bool
}
