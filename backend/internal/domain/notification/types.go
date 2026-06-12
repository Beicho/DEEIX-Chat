package notification

import "time"

const (
	// TypeAnnouncement 表示公告通知。
	TypeAnnouncement = "announcement"
	// SourceAnnouncement 表示通知来自公告系统。
	SourceAnnouncement = "announcement"
)

// Notification 表示一条持久化站内通知。
type Notification struct {
	ID        uint
	UserID    uint
	Type      string
	Title     string
	Body      string
	Link      string
	ReadAt    *time.Time
	Source    string
	SourceID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
