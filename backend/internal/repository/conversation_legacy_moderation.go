package repository

import (
	"context"
	"time"

	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
)

// ConversationLegacyModerationRepository preserves the production status page
// and moderation-event administration contracts while the upstream moderation
// coordinator is adopted independently.
type ConversationLegacyModerationRepository interface {
	ListModelAvailability(ctx context.Context, since time.Time) ([]domainconversation.ModelAvailability, error)
	CreateModerationEvent(ctx context.Context, item *domainconversation.ModerationEvent) error
	ListModerationEvents(ctx context.Context, filter domainconversation.ModerationEventFilter, offset int, limit int) ([]domainconversation.ModerationEvent, int64, error)
	CountFlaggedModerationEvents(ctx context.Context, userID uint, since time.Time) (int64, error)
	GetModerationEvent(ctx context.Context, id uint) (*domainconversation.ModerationEvent, error)
	UpdateModerationEventReview(ctx context.Context, id uint, status string, reviewedBy uint, reviewedAt *time.Time, note string) (*domainconversation.ModerationEvent, error)
	UpdateModerationEventDisposition(ctx context.Context, id uint, disposition string, appliedAt *time.Time) (*domainconversation.ModerationEvent, error)
	ReleaseModerationEventDisposition(ctx context.Context, id uint, reviewerID uint, releasedAt *time.Time) (*domainconversation.ModerationEvent, error)
}
