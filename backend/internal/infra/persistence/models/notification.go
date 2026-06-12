package model

import "time"

// Notification 记录用户站内通知。
type Notification struct {
	BaseModel
	UserID   uint       `gorm:"not null;index:idx_notifications_user_read,priority:1;index:idx_notifications_source,priority:1;comment:接收用户ID"`
	Type     string     `gorm:"size:64;not null;default:'';index:idx_notifications_type;comment:通知类型"`
	Title    string     `gorm:"size:200;not null;default:'';comment:通知标题"`
	Body     string     `gorm:"type:text;not null;default:'';comment:通知正文"`
	Link     string     `gorm:"size:500;not null;default:'';comment:跳转链接"`
	ReadAt   *time.Time `gorm:"index:idx_notifications_user_read,priority:2;comment:已读时间"`
	Source   string     `gorm:"size:64;not null;default:'';index:idx_notifications_source,priority:2;comment:来源系统"`
	SourceID string     `gorm:"size:128;not null;default:'';index:idx_notifications_source,priority:3;comment:来源记录ID"`
}

// TableName 指定表名。
func (Notification) TableName() string {
	return "notifications"
}
