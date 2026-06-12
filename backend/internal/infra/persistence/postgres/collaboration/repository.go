package collaboration

import (
	"context"
	"errors"
	"strings"

	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repo implements collaboration persistence.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates a collaboration repository.
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateAssistant(ctx context.Context, item *domaincollab.Assistant) error {
	if item == nil || item.OwnerUserID == 0 {
		return repository.ErrInvalidInput
	}
	record := assistantToRecord(*item)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return translateError(err)
	}
	*item = assistantToDomain(record, false)
	return nil
}

func (r *Repo) ListAssistants(ctx context.Context, userID uint, includePublic bool, offset int, limit int) ([]domaincollab.Assistant, int64, error) {
	if userID == 0 {
		return nil, 0, repository.ErrInvalidInput
	}
	query := r.db.WithContext(ctx).Model(&model.Assistant{}).Where("status <> ?", "deleted")
	if includePublic {
		query = query.Where("owner_user_id = ? OR visibility = ?", userID, domaincollab.AssistantVisibilityPublic)
	} else {
		query = query.Where("owner_user_id = ?", userID)
	}
	return r.listAssistants(ctx, query, userID, offset, limit)
}

func (r *Repo) ListPublicAssistants(ctx context.Context, userID uint, offset int, limit int) ([]domaincollab.Assistant, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Assistant{}).
		Where("status <> ? AND visibility = ?", "deleted", domaincollab.AssistantVisibilityPublic)
	return r.listAssistants(ctx, query, userID, offset, limit)
}

func (r *Repo) listAssistants(ctx context.Context, query *gorm.DB, userID uint, offset int, limit int) ([]domaincollab.Assistant, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	records := make([]model.Assistant, 0, limit)
	if err := query.Order("CASE WHEN published_at IS NULL THEN 1 ELSE 0 END ASC, published_at DESC, updated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, translateError(err)
	}
	installed := map[uint]bool{}
	if userID > 0 && len(records) > 0 {
		ids := make([]uint, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		var installs []model.AssistantInstall
		if err := r.db.WithContext(ctx).Where("user_id = ? AND assistant_id IN ?", userID, ids).Find(&installs).Error; err != nil {
			return nil, 0, translateError(err)
		}
		for _, install := range installs {
			installed[install.AssistantID] = true
		}
	}
	results := make([]domaincollab.Assistant, 0, len(records))
	for _, record := range records {
		results = append(results, assistantToDomain(record, installed[record.ID]))
	}
	return results, total, nil
}

func (r *Repo) GetAssistantByPublicID(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error) {
	record, err := r.findVisibleAssistant(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	installed := false
	if userID > 0 {
		var count int64
		if err := r.db.WithContext(ctx).Model(&model.AssistantInstall{}).Where("user_id = ? AND assistant_id = ?", userID, record.ID).Count(&count).Error; err != nil {
			return nil, translateError(err)
		}
		installed = count > 0
	}
	item := assistantToDomain(*record, installed)
	return &item, nil
}

func (r *Repo) UpdateAssistant(ctx context.Context, userID uint, publicID string, patch repository.AssistantPatch) (*domaincollab.Assistant, error) {
	if userID == 0 || strings.TrimSpace(publicID) == "" {
		return nil, repository.ErrInvalidInput
	}
	updates := map[string]interface{}{}
	if patch.Name != nil {
		updates["name"] = *patch.Name
	}
	if patch.AvatarURL != nil {
		updates["avatar_url"] = *patch.AvatarURL
	}
	if patch.Description != nil {
		updates["description"] = *patch.Description
	}
	if patch.SystemPrompt != nil {
		updates["system_prompt"] = *patch.SystemPrompt
	}
	if patch.DefaultModel != nil {
		updates["default_model"] = *patch.DefaultModel
	}
	if patch.OpeningMessage != nil {
		updates["opening_message"] = *patch.OpeningMessage
	}
	if patch.Visibility != nil {
		updates["visibility"] = *patch.Visibility
	}
	if patch.Status != nil {
		updates["status"] = *patch.Status
	}
	if patch.PublishedAt != nil {
		updates["published_at"] = *patch.PublishedAt
	}
	if len(updates) == 0 {
		return r.GetAssistantByPublicID(ctx, userID, publicID)
	}
	result := r.db.WithContext(ctx).Model(&model.Assistant{}).
		Where("public_id = ? AND owner_user_id = ? AND status <> ?", strings.TrimSpace(publicID), userID, "deleted").
		Updates(updates)
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.GetAssistantByPublicID(ctx, userID, publicID)
}

func (r *Repo) DeleteAssistant(ctx context.Context, userID uint, publicID string) error {
	result := r.db.WithContext(ctx).Model(&model.Assistant{}).
		Where("public_id = ? AND owner_user_id = ?", strings.TrimSpace(publicID), userID).
		Update("status", "deleted")
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repo) InstallAssistant(ctx context.Context, userID uint, assistantPublicID string) (*domaincollab.AssistantInstall, error) {
	record, err := r.findVisibleAssistant(ctx, userID, assistantPublicID)
	if err != nil {
		return nil, err
	}
	install := model.AssistantInstall{UserID: userID, AssistantID: record.ID}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&install).Error; err != nil {
		return nil, translateError(err)
	}
	var saved model.AssistantInstall
	if err := r.db.WithContext(ctx).Where("user_id = ? AND assistant_id = ?", userID, record.ID).First(&saved).Error; err != nil {
		return nil, translateError(err)
	}
	result := domaincollab.AssistantInstall{ID: saved.ID, UserID: saved.UserID, AssistantID: saved.AssistantID, CreatedAt: saved.CreatedAt, UpdatedAt: saved.UpdatedAt}
	return &result, nil
}

func (r *Repo) UninstallAssistant(ctx context.Context, userID uint, assistantPublicID string) error {
	record, err := r.findVisibleAssistant(ctx, userID, assistantPublicID)
	if err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Where("user_id = ? AND assistant_id = ?", userID, record.ID).Delete(&model.AssistantInstall{})
	return translateError(result.Error)
}

func (r *Repo) GetInstalledAssistantPrompt(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error) {
	record, err := r.findVisibleAssistant(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	if record.OwnerUserID != userID {
		var count int64
		if err := r.db.WithContext(ctx).Model(&model.AssistantInstall{}).Where("user_id = ? AND assistant_id = ?", userID, record.ID).Count(&count).Error; err != nil {
			return nil, translateError(err)
		}
		if count == 0 {
			return nil, repository.ErrNotFound
		}
	}
	item := assistantToDomain(*record, true)
	return &item, nil
}

func (r *Repo) findVisibleAssistant(ctx context.Context, userID uint, publicID string) (*model.Assistant, error) {
	if strings.TrimSpace(publicID) == "" {
		return nil, repository.ErrInvalidInput
	}
	var record model.Assistant
	err := r.db.WithContext(ctx).
		Where("public_id = ? AND status <> ? AND (owner_user_id = ? OR visibility = ?)", strings.TrimSpace(publicID), "deleted", userID, domaincollab.AssistantVisibilityPublic).
		First(&record).Error
	if err != nil {
		return nil, translateError(err)
	}
	return &record, nil
}

func assistantToRecord(item domaincollab.Assistant) model.Assistant {
	return model.Assistant{
		PublicID:       item.PublicID,
		OwnerUserID:    item.OwnerUserID,
		Name:           item.Name,
		AvatarURL:      item.AvatarURL,
		Description:    item.Description,
		SystemPrompt:   item.SystemPrompt,
		DefaultModel:   item.DefaultModel,
		OpeningMessage: item.OpeningMessage,
		Visibility:     item.Visibility,
		Status:         item.Status,
		PublishedAt:    item.PublishedAt,
	}
}

func assistantToDomain(item model.Assistant, installed bool) domaincollab.Assistant {
	return domaincollab.Assistant{
		ID:             item.ID,
		PublicID:       item.PublicID,
		OwnerUserID:    item.OwnerUserID,
		Name:           item.Name,
		AvatarURL:      item.AvatarURL,
		Description:    item.Description,
		SystemPrompt:   item.SystemPrompt,
		DefaultModel:   item.DefaultModel,
		OpeningMessage: item.OpeningMessage,
		Visibility:     item.Visibility,
		Status:         item.Status,
		PublishedAt:    item.PublishedAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		Installed:      installed,
	}
}

func translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}
	return err
}
