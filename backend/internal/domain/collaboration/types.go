package collaboration

import "time"

const (
	AssistantVisibilityPrivate = "private"
	AssistantVisibilityPublic  = "public"

	TeamRoleOwner  = "owner"
	TeamRoleAdmin  = "admin"
	TeamRoleMember = "member"
)

// Assistant describes a user-owned assistant preset.
type Assistant struct {
	ID             uint
	PublicID       string
	OwnerUserID    uint
	Name           string
	AvatarURL      string
	Description    string
	SystemPrompt   string
	DefaultModel   string
	OpeningMessage string
	Visibility     string
	Status         string
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Installed      bool
}

// AssistantInstall links a marketplace assistant to a user.
type AssistantInstall struct {
	ID          uint
	UserID      uint
	AssistantID uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ScheduledPrompt stores one due prompt reminder.
type ScheduledPrompt struct {
	ID                         uint
	PublicID                   string
	UserID                     uint
	AssistantID                *uint
	AssistantPublicID          string
	TargetConversationID       *uint
	TargetConversationPublicID string
	TargetConversationTitle    string
	Title                      string
	Content                    string
	DueAt                      time.Time
	NextRunAt                  time.Time
	ScheduleType               string
	ScheduleTime               string
	ScheduleWeekday            int
	CronExpression             string
	Model                      string
	Enabled                    bool
	Status                     string
	LastTriggeredAt            *time.Time
	RetryCount                 int
	LastError                  string
	ConversationID             *uint
	ConversationSlug           string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// TeamSpace is a lightweight shared workspace container.
type TeamSpace struct {
	ID          uint
	PublicID    string
	OwnerUserID uint
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Members     []TeamMember
}

// TeamMember links users to a team.
type TeamMember struct {
	ID          uint
	TeamID      uint
	UserID      uint
	Role        string
	InvitedBy   uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Username    string
	DisplayName string
	AvatarURL   string
	Email       string
}
