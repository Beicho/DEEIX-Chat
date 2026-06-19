package conversation

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// ListModelAvailability returns public model health aggregated from recent run logs.
func (s *Service) ListModelAvailability(ctx context.Context, window time.Duration) ([]model.ModelAvailability, error) {
	if window <= 0 {
		window = 24 * time.Hour
	}
	return s.repo.ListModelAvailability(ctx, time.Now().Add(-window))
}

// ListModerationEvents returns recent moderation events for admin review.
func (s *Service) ListModerationEvents(ctx context.Context, page int, pageSize int, filter model.ModerationEventFilter) ([]model.ModerationEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	filter.Direction = normalizeModerationListOption(filter.Direction, "input", "output")
	filter.ReviewStatus = normalizeModerationListOption(filter.ReviewStatus, "pending", "false_positive", "confirmed", "resolved")
	filter.Disposition = normalizeModerationListOption(filter.Disposition, "none", moderationDispositionLimit, moderationDispositionSuspend)
	filter.EventType = normalizeModerationListOption(filter.EventType, moderationEventPolicyHit, moderationEventEngineError)
	return s.repo.ListModerationEvents(ctx, filter, (page-1)*pageSize, pageSize)
}

func normalizeModerationListOption(value string, allowed ...string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return ""
	}
	for _, option := range allowed {
		if normalized == option {
			return normalized
		}
	}
	return ""
}

// UpdateModerationReview marks one moderation event review state.
func (s *Service) UpdateModerationReview(ctx context.Context, id uint, reviewerID uint, status string, note string) (*model.ModerationEvent, error) {
	normalized := strings.TrimSpace(status)
	switch normalized {
	case "pending", "false_positive", "confirmed", "resolved":
	default:
		return nil, ErrInvalidMessageContent
	}
	now := time.Now().UTC()
	item, err := s.repo.UpdateModerationEventReview(ctx, id, normalized, reviewerID, &now, note)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return item, nil
}

// ReleaseModerationDisposition clears account status changes created by moderation automation.
func (s *Service) ReleaseModerationDisposition(ctx context.Context, id uint, reviewerID uint, note string) (*model.ModerationEvent, error) {
	item, err := s.repo.GetModerationEvent(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	if strings.TrimSpace(item.Disposition) == "" || item.DispositionReleasedAt != nil {
		return nil, ErrInvalidMessageContent
	}
	if item.Disposition != moderationDispositionLimit && item.Disposition != moderationDispositionSuspend {
		return nil, ErrInvalidMessageContent
	}
	now := time.Now().UTC()
	if item.Disposition == moderationDispositionLimit && s.rateLimiter != nil && item.UserID != 0 {
		if err = s.rateLimiter.ClearUserRateLimitOverride(ctx, item.UserID); err != nil {
			return nil, err
		}
	}
	if s.userEnforcer != nil && item.UserID != 0 {
		user, userErr := s.userEnforcer.GetByID(ctx, item.UserID)
		if userErr != nil {
			return nil, userErr
		}
		if user != nil && !domainuser.IsAdminRole(user.Role) && (user.Status == domainuser.StatusLocked || user.Status == domainuser.StatusSuspended) {
			if !isModerationOwnedAccountDisposition(user) {
				return nil, ErrInvalidMessageContent
			}
			if err = s.userEnforcer.UpdateUserStatus(ctx, item.UserID, domainuser.StatusActive); err != nil {
				return nil, err
			}
			if err = s.userEnforcer.SetUserSuspension(ctx, item.UserID, "", "", nil, nil); err != nil {
				return nil, err
			}
		}
	}
	if _, err = s.repo.ReleaseModerationEventDisposition(ctx, id, reviewerID, &now); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	updated, err := s.UpdateModerationReview(ctx, id, reviewerID, "resolved", note)
	if err != nil {
		return nil, err
	}
	s.notifyModerationDispositionRelease(ctx, item.UserID, item.ID, item.Disposition, now)
	return updated, nil
}

func isModerationOwnedAccountDisposition(user *domainuser.User) bool {
	if user == nil {
		return false
	}
	reason := strings.TrimSpace(user.SuspensionReason)
	detail := strings.TrimSpace(user.SuspensionDetail)
	if reason == moderationVisibleSuspensionReason {
		return detail == moderationAutoLimitDetail || detail == moderationAutoSuspendDetail
	}
	if reason == "content_policy" {
		return detail == moderationAutoLimitRevokeReason || detail == moderationAutoSuspendRevokeReason ||
			detail == "content_policy_auto_limit" || detail == "content_policy_auto_suspension"
	}
	return false
}
