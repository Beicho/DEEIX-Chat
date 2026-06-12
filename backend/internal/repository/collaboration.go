package repository

import (
	"context"
	"time"

	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
)

// CollaborationRepository defines assistant, scheduling, and team persistence.
type CollaborationRepository interface {
	CreateAssistant(ctx context.Context, item *domaincollab.Assistant) error
	ListAssistants(ctx context.Context, userID uint, includePublic bool, offset int, limit int) ([]domaincollab.Assistant, int64, error)
	ListPublicAssistants(ctx context.Context, userID uint, offset int, limit int) ([]domaincollab.Assistant, int64, error)
	GetAssistantByPublicID(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error)
	UpdateAssistant(ctx context.Context, userID uint, publicID string, patch AssistantPatch) (*domaincollab.Assistant, error)
	DeleteAssistant(ctx context.Context, userID uint, publicID string) error
	InstallAssistant(ctx context.Context, userID uint, assistantPublicID string) (*domaincollab.AssistantInstall, error)
	UninstallAssistant(ctx context.Context, userID uint, assistantPublicID string) error
	GetInstalledAssistantPrompt(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error)

	CreateScheduledPrompt(ctx context.Context, item *domaincollab.ScheduledPrompt) error
	ListScheduledPrompts(ctx context.Context, userID uint, offset int, limit int) ([]domaincollab.ScheduledPrompt, int64, error)
	UpdateScheduledPrompt(ctx context.Context, userID uint, publicID string, patch ScheduledPromptPatch) (*domaincollab.ScheduledPrompt, error)
	DeleteScheduledPrompt(ctx context.Context, userID uint, publicID string) error
	ListDueScheduledPrompts(ctx context.Context, now time.Time, limit int) ([]domaincollab.ScheduledPrompt, error)
	GetConversationTarget(ctx context.Context, userID uint, publicID string) (uint, string, error)
	MarkScheduledPromptSucceeded(ctx context.Context, id uint, now time.Time, nextRunAt *time.Time, conversationID uint) error
	MarkScheduledPromptFailed(ctx context.Context, id uint, lastError string, retryAt *time.Time, disable bool) error

	CreateTeamSpace(ctx context.Context, team *domaincollab.TeamSpace, owner *domaincollab.TeamMember) error
	ListTeamSpaces(ctx context.Context, userID uint) ([]domaincollab.TeamSpace, error)
	GetTeamSpaceByPublicID(ctx context.Context, userID uint, publicID string) (*domaincollab.TeamSpace, error)
	AddTeamMember(ctx context.Context, teamPublicID string, actorUserID uint, memberUserID uint, role string) (*domaincollab.TeamMember, error)
	RemoveTeamMember(ctx context.Context, teamPublicID string, actorUserID uint, memberUserID uint) error
	FindUserByLogin(ctx context.Context, login string) (*domainuser.User, error)
}

// AssistantPatch describes assistant fields that may be updated.
type AssistantPatch struct {
	Name           *string
	AvatarURL      *string
	Description    *string
	SystemPrompt   *string
	DefaultModel   *string
	OpeningMessage *string
	Visibility     *string
	Status         *string
	PublishedAt    *time.Time
}

// ScheduledPromptPatch describes scheduled prompt fields that may be updated.
type ScheduledPromptPatch struct {
	AssistantID          **uint
	TargetConversationID **uint
	Title                *string
	Content              *string
	DueAt                *time.Time
	NextRunAt            *time.Time
	ScheduleType         *string
	ScheduleTime         *string
	ScheduleWeekday      *int
	CronExpression       *string
	Model                *string
	Enabled              *bool
	Status               *string
	LastTriggeredAt      *time.Time
	RetryCount           *int
	LastError            *string
}
