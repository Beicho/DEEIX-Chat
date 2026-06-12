package notification

import "errors"

var (
	// ErrInvalidNotification 表示通知请求参数非法。
	ErrInvalidNotification = errors.New("invalid notification")
	// ErrNotificationNotFound 表示通知不存在。
	ErrNotificationNotFound = errors.New("notification not found")
)
