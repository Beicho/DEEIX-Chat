package conversation

import (
	"context"
	"strings"
	"time"

	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	models "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
)

// ListModelAvailability 汇总公开状态页所需的模型调用结果，不返回渠道或来源信息。
func (r *Repo) ListModelAvailability(ctx context.Context, since time.Time) ([]domainconversation.ModelAvailability, error) {
	type row struct {
		ModelName string  `gorm:"column:model_name"`
		CallCount int64   `gorm:"column:call_count"`
		Successes int64   `gorm:"column:successes"`
		Rate      float64 `gorm:"column:success_rate"`
	}
	rows := make([]row, 0)
	err := r.db.WithContext(ctx).
		Model(&models.ConversationRun{}).
		Select(`
			COALESCE(NULLIF(platform_model_name, ''), NULLIF(requested_model_name, '')) AS model_name,
			COUNT(*) AS call_count,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successes,
			CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS float) / NULLIF(COUNT(*), 0) AS success_rate
		`).
		Where("task_type = ? AND started_at >= ?", "chat", since).
		Where("(platform_model_name <> '' OR requested_model_name <> '')").
		Group("model_name").
		Order("model_name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, translateError(err)
	}
	results := make([]domainconversation.ModelAvailability, 0, len(rows))
	for _, item := range rows {
		status := "normal"
		switch {
		case item.CallCount > 0 && item.Rate < 0.5:
			status = "down"
		case item.CallCount > 0 && item.Rate < 0.95:
			status = "degraded"
		}
		results = append(results, domainconversation.ModelAvailability{
			ModelName:   strings.TrimSpace(item.ModelName),
			CallCount:   item.CallCount,
			SuccessRate: item.Rate,
			Status:      status,
		})
	}
	return results, nil
}

// CreateModerationEvent 写入内容检查事件。
func (r *Repo) CreateModerationEvent(ctx context.Context, item *domainconversation.ModerationEvent) error {
	if item == nil {
		return nil
	}
	entity := toModerationEventModel(item)
	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return translateError(err)
	}
	*item = toModerationEventDomain(entity)
	return nil
}

// ListModerationEvents 分页列出内容检查事件。
func (r *Repo) ListModerationEvents(ctx context.Context, filter domainconversation.ModerationEventFilter, offset int, limit int) ([]domainconversation.ModerationEvent, int64, error) {
	items := make([]models.ModerationEvent, 0)
	var total int64
	query := applyModerationEventFilter(r.db.WithContext(ctx).Model(&models.ModerationEvent{}), filter)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainconversation.ModerationEvent, 0, len(items))
	for _, item := range items {
		results = append(results, toModerationEventDomain(item))
	}
	return results, total, nil
}

func applyModerationEventFilter(query *gorm.DB, filter domainconversation.ModerationEventFilter) *gorm.DB {
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if value := strings.TrimSpace(filter.Direction); value != "" {
		query = query.Where("direction = ?", value)
	}
	if value := strings.TrimSpace(filter.ReviewStatus); value != "" {
		query = query.Where("review_status = ?", value)
	}
	if value := strings.TrimSpace(filter.Disposition); value != "" {
		if value == "none" {
			query = query.Where("disposition = ''")
		} else {
			query = query.Where("disposition = ?", value)
		}
	}
	if value := strings.TrimSpace(filter.EventType); value != "" {
		query = query.Where("event_type = ?", value)
	}
	if filter.Flagged != nil {
		query = query.Where("flagged = ?", *filter.Flagged)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	return query
}

// CountFlaggedModerationEvents counts flagged content checks for one user since a point in time.
func (r *Repo) CountFlaggedModerationEvents(ctx context.Context, userID uint, since time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&models.ModerationEvent{}).
		Where("user_id = ? AND flagged = ? AND reason = ? AND created_at >= ?", userID, true, "blocked", since).
		Count(&total).Error
	return total, translateError(err)
}

// GetModerationEvent returns one moderation event by numeric ID.
func (r *Repo) GetModerationEvent(ctx context.Context, id uint) (*domainconversation.ModerationEvent, error) {
	var item models.ModerationEvent
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, translateError(err)
	}
	result := toModerationEventDomain(item)
	return &result, nil
}

// UpdateModerationEventReview stores admin review metadata for an event.
func (r *Repo) UpdateModerationEventReview(ctx context.Context, id uint, status string, reviewedBy uint, reviewedAt *time.Time, note string) (*domainconversation.ModerationEvent, error) {
	updates := map[string]interface{}{
		"review_status": strings.TrimSpace(status),
		"reviewed_by":   reviewedBy,
		"reviewed_at":   reviewedAt,
		"review_note":   strings.TrimSpace(note),
	}
	result := r.db.WithContext(ctx).Model(&models.ModerationEvent{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.GetModerationEvent(ctx, id)
}

// UpdateModerationEventDisposition records the automatic moderation disposition attached to one event.
func (r *Repo) UpdateModerationEventDisposition(ctx context.Context, id uint, disposition string, appliedAt *time.Time) (*domainconversation.ModerationEvent, error) {
	updates := map[string]interface{}{
		"disposition":            strings.TrimSpace(disposition),
		"disposition_applied_at": appliedAt,
	}
	result := r.db.WithContext(ctx).Model(&models.ModerationEvent{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.GetModerationEvent(ctx, id)
}

// ReleaseModerationEventDisposition records who released an automatic moderation disposition.
func (r *Repo) ReleaseModerationEventDisposition(ctx context.Context, id uint, reviewerID uint, releasedAt *time.Time) (*domainconversation.ModerationEvent, error) {
	updates := map[string]interface{}{
		"disposition_released_by": reviewerID,
		"disposition_released_at": releasedAt,
	}
	result := r.db.WithContext(ctx).Model(&models.ModerationEvent{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.GetModerationEvent(ctx, id)
}

func toModerationEventDomain(item models.ModerationEvent) domainconversation.ModerationEvent {
	return domainconversation.ModerationEvent{
		ID:                    item.ID,
		UserID:                item.UserID,
		ConversationID:        item.ConversationID,
		MessageID:             item.MessageID,
		RunID:                 item.RunID,
		Direction:             item.Direction,
		Action:                item.Action,
		Model:                 item.Model,
		Score:                 item.Score,
		Threshold:             item.Threshold,
		Flagged:               item.Flagged,
		CategoriesJSON:        item.CategoriesJSON,
		Reason:                item.Reason,
		EventType:             item.EventType,
		ContentSnapshot:       item.ContentSnapshot,
		ContentHash:           item.ContentHash,
		SnapshotTruncated:     item.SnapshotTruncated,
		ReviewStatus:          item.ReviewStatus,
		ReviewedBy:            item.ReviewedBy,
		ReviewedAt:            item.ReviewedAt,
		ReviewNote:            item.ReviewNote,
		Disposition:           item.Disposition,
		DispositionAppliedAt:  item.DispositionAppliedAt,
		DispositionReleasedAt: item.DispositionReleasedAt,
		DispositionReleasedBy: item.DispositionReleasedBy,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func toModerationEventModel(item *domainconversation.ModerationEvent) models.ModerationEvent {
	if item == nil {
		return models.ModerationEvent{}
	}
	return models.ModerationEvent{
		UserID:                item.UserID,
		ConversationID:        item.ConversationID,
		MessageID:             item.MessageID,
		RunID:                 item.RunID,
		Direction:             item.Direction,
		Action:                item.Action,
		Model:                 item.Model,
		Score:                 item.Score,
		Threshold:             item.Threshold,
		Flagged:               item.Flagged,
		CategoriesJSON:        item.CategoriesJSON,
		Reason:                item.Reason,
		EventType:             item.EventType,
		ContentSnapshot:       item.ContentSnapshot,
		ContentHash:           item.ContentHash,
		SnapshotTruncated:     item.SnapshotTruncated,
		ReviewStatus:          item.ReviewStatus,
		ReviewedBy:            item.ReviewedBy,
		ReviewedAt:            item.ReviewedAt,
		ReviewNote:            item.ReviewNote,
		Disposition:           item.Disposition,
		DispositionAppliedAt:  item.DispositionAppliedAt,
		DispositionReleasedAt: item.DispositionReleasedAt,
		DispositionReleasedBy: item.DispositionReleasedBy,
	}
}
