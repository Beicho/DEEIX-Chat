package notification

import "time"

const (
	// TypeSystem 表示系统通知。
	TypeSystem = "system"
	// TypeAnnouncement 表示公告通知。
	TypeAnnouncement = "announcement"
	// TypeAuth 表示账号安全通知。
	TypeAuth = "auth"
	// TypeBillingExpiry 表示订阅到期提醒。
	TypeBillingExpiry = "billing_expiry"
	// TypeModeration 表示内容安全处置通知。
	TypeModeration = "moderation"
	// TypeWeeklySummary 表示使用周报通知。
	TypeWeeklySummary = "weekly_summary"
	// TypeScheduledPrompt 表示定时任务通知。
	TypeScheduledPrompt = "scheduled_prompt"
	// SourceSystem 表示通知来自系统内部。
	SourceSystem = "system"
	// SourceAnnouncement 表示通知来自公告系统。
	SourceAnnouncement = "announcement"
	// SourceScheduledPrompt 表示通知来自定时任务系统。
	SourceScheduledPrompt = "scheduled_prompt"
)

// Notification 表示一条持久化站内通知。
type Notification struct {
	ID        uint
	UserID    uint
	Type      string
	Title     string
	Body      string
	ActionURL string
	Link      string
	ReadAt    *time.Time
	Source    string
	SourceID  string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}
