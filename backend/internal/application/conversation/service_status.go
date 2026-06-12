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
func (s *Service) ListModerationEvents(ctx context.Context, page int, pageSize int) ([]model.ModerationEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.repo.ListModerationEvents(ctx, (page-1)*pageSize, pageSize)
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
	if s.userEnforcer != nil && item.UserID != 0 {
		user, userErr := s.userEnforcer.GetByID(ctx, item.UserID)
		if userErr != nil {
			return nil, userErr
		}
		if user != nil && !domainuser.IsAdminRole(user.Role) && (user.Status == domainuser.StatusLocked || user.Status == domainuser.StatusSuspended) {
			if err = s.userEnforcer.UpdateUserStatus(ctx, item.UserID, domainuser.StatusActive); err != nil {
				return nil, err
			}
			if err = s.userEnforcer.SetUserSuspension(ctx, item.UserID, "", "", nil, nil); err != nil {
				return nil, err
			}
		}
	}
	return s.UpdateModerationReview(ctx, id, reviewerID, "resolved", note)
}
