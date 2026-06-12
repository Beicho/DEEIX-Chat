package collaboration

import (
	"context"
	"strings"
	"time"

	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repo) CreateScheduledPrompt(ctx context.Context, item *domaincollab.ScheduledPrompt) error {
	if item == nil || item.UserID == 0 {
		return repository.ErrInvalidInput
	}
	record := scheduledPromptToRecord(*item)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return translateError(err)
	}
	*item = r.scheduledPromptToDomainWithTarget(ctx, record)
	return nil
}

func (r *Repo) ListScheduledPrompts(ctx context.Context, userID uint, offset int, limit int) ([]domaincollab.ScheduledPrompt, int64, error) {
	if userID == 0 {
		return nil, 0, repository.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	query := r.db.WithContext(ctx).Model(&model.ScheduledPrompt{}).Where("user_id = ? AND status <> ?", userID, "deleted")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	var records []model.ScheduledPrompt
	if err := query.Order("due_at ASC, id DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domaincollab.ScheduledPrompt, 0, len(records))
	for _, record := range records {
		results = append(results, r.scheduledPromptToDomainWithTarget(ctx, record))
	}
	return results, total, nil
}

func (r *Repo) UpdateScheduledPrompt(ctx context.Context, userID uint, publicID string, patch repository.ScheduledPromptPatch) (*domaincollab.ScheduledPrompt, error) {
	updates := map[string]interface{}{}
	if patch.AssistantID != nil {
		updates["assistant_id"] = *patch.AssistantID
	}
	if patch.TargetConversationID != nil {
		updates["target_conversation_id"] = *patch.TargetConversationID
	}
	if patch.Title != nil {
		updates["title"] = *patch.Title
	}
	if patch.Content != nil {
		updates["content"] = *patch.Content
	}
	if patch.DueAt != nil {
		updates["due_at"] = *patch.DueAt
	}
	if patch.NextRunAt != nil {
		updates["next_run_at"] = *patch.NextRunAt
	}
	if patch.ScheduleType != nil {
		updates["schedule_type"] = *patch.ScheduleType
	}
	if patch.ScheduleTime != nil {
		updates["schedule_time"] = *patch.ScheduleTime
	}
	if patch.ScheduleWeekday != nil {
		updates["schedule_weekday"] = *patch.ScheduleWeekday
	}
	if patch.CronExpression != nil {
		updates["cron_expression"] = *patch.CronExpression
	}
	if patch.Model != nil {
		updates["model"] = *patch.Model
	}
	if patch.Enabled != nil {
		updates["enabled"] = *patch.Enabled
	}
	if patch.Status != nil {
		updates["status"] = *patch.Status
	}
	if patch.LastTriggeredAt != nil {
		updates["last_triggered_at"] = *patch.LastTriggeredAt
	}
	if patch.RetryCount != nil {
		updates["retry_count"] = *patch.RetryCount
	}
	if patch.LastError != nil {
		updates["last_error"] = *patch.LastError
	}
	if len(updates) == 0 {
		return r.getScheduledPrompt(ctx, userID, publicID)
	}
	result := r.db.WithContext(ctx).Model(&model.ScheduledPrompt{}).
		Where("public_id = ? AND user_id = ? AND status <> ?", strings.TrimSpace(publicID), userID, "deleted").
		Updates(updates)
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.getScheduledPrompt(ctx, userID, publicID)
}

func (r *Repo) DeleteScheduledPrompt(ctx context.Context, userID uint, publicID string) error {
	result := r.db.WithContext(ctx).Model(&model.ScheduledPrompt{}).
		Where("public_id = ? AND user_id = ?", strings.TrimSpace(publicID), userID).
		Updates(map[string]interface{}{"status": "deleted", "enabled": false})
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repo) ListDueScheduledPrompts(ctx context.Context, now time.Time, limit int) ([]domaincollab.ScheduledPrompt, error) {
	if limit <= 0 {
		limit = 50
	}
	var records []model.ScheduledPrompt
	if err := r.db.WithContext(ctx).
		Where("enabled = ? AND status = ? AND next_run_at <= ?", true, "scheduled", now).
		Order("next_run_at ASC, id ASC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domaincollab.ScheduledPrompt, 0, len(records))
	for _, record := range records {
		results = append(results, r.scheduledPromptToDomainWithTarget(ctx, record))
	}
	return results, nil
}

func (r *Repo) GetConversationTarget(ctx context.Context, userID uint, publicID string) (uint, string, error) {
	var item model.Conversation
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND public_id = ? AND status <> ?", userID, strings.TrimSpace(publicID), "deleted").
		First(&item).Error; err != nil {
		return 0, "", translateError(err)
	}
	return item.ID, item.Title, nil
}

func (r *Repo) MarkScheduledPromptSucceeded(ctx context.Context, id uint, now time.Time, nextRunAt *time.Time, conversationID uint) error {
	updates := map[string]interface{}{
		"last_triggered_at": now,
		"retry_count":       0,
		"last_error":        "",
		"conversation_id":   conversationID,
	}
	if nextRunAt == nil || nextRunAt.IsZero() {
		updates["enabled"] = false
		updates["status"] = "triggered"
	} else {
		updates["next_run_at"] = *nextRunAt
		updates["due_at"] = *nextRunAt
		updates["status"] = "scheduled"
	}
	result := r.db.WithContext(ctx).Model(&model.ScheduledPrompt{}).
		Where("id = ? AND status = ?", id, "scheduled").
		Updates(updates)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repo) MarkScheduledPromptFailed(ctx context.Context, id uint, lastError string, retryAt *time.Time, disable bool) error {
	updates := map[string]interface{}{
		"last_error":  strings.TrimSpace(lastError),
		"retry_count": gorm.Expr("retry_count + 1"),
	}
	if disable {
		updates["enabled"] = false
		updates["status"] = "failed"
	} else if retryAt != nil && !retryAt.IsZero() {
		updates["next_run_at"] = *retryAt
		updates["due_at"] = *retryAt
	}
	result := r.db.WithContext(ctx).Model(&model.ScheduledPrompt{}).
		Where("id = ? AND status = ?", id, "scheduled").
		Updates(updates)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repo) getScheduledPrompt(ctx context.Context, userID uint, publicID string) (*domaincollab.ScheduledPrompt, error) {
	var record model.ScheduledPrompt
	if err := r.db.WithContext(ctx).Where("public_id = ? AND user_id = ? AND status <> ?", strings.TrimSpace(publicID), userID, "deleted").First(&record).Error; err != nil {
		return nil, translateError(err)
	}
	item := r.scheduledPromptToDomainWithTarget(ctx, record)
	return &item, nil
}

func (r *Repo) scheduledPromptToDomainWithTarget(ctx context.Context, record model.ScheduledPrompt) domaincollab.ScheduledPrompt {
	item := scheduledPromptToDomain(record, "")
	if record.TargetConversationID == nil || *record.TargetConversationID == 0 {
		return item
	}
	var conversation model.Conversation
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", *record.TargetConversationID, record.UserID).First(&conversation).Error; err == nil {
		item.TargetConversationPublicID = conversation.PublicID
		item.TargetConversationTitle = conversation.Title
		item.ConversationSlug = conversation.PublicID
	}
	return item
}

func (r *Repo) CreateTeamSpace(ctx context.Context, team *domaincollab.TeamSpace, owner *domaincollab.TeamMember) error {
	if team == nil || owner == nil || team.OwnerUserID == 0 || owner.UserID == 0 {
		return repository.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		teamRecord := teamToRecord(*team)
		if err := tx.Create(&teamRecord).Error; err != nil {
			return translateError(err)
		}
		memberRecord := model.TeamMember{TeamID: teamRecord.ID, UserID: owner.UserID, Role: owner.Role, InvitedBy: owner.InvitedBy}
		if err := tx.Create(&memberRecord).Error; err != nil {
			return translateError(err)
		}
		*team = teamToDomain(teamRecord, nil)
		*owner = memberToDomain(memberRecord, model.User{})
		return nil
	})
}

func (r *Repo) ListTeamSpaces(ctx context.Context, userID uint) ([]domaincollab.TeamSpace, error) {
	if userID == 0 {
		return nil, repository.ErrInvalidInput
	}
	var teams []model.TeamSpace
	if err := r.db.WithContext(ctx).
		Joins("JOIN team_members ON team_members.team_id = team_spaces.id AND team_members.deleted_at IS NULL").
		Where("team_members.user_id = ? AND team_spaces.status <> ?", userID, "deleted").
		Order("team_spaces.updated_at DESC, team_spaces.id DESC").
		Find(&teams).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domaincollab.TeamSpace, 0, len(teams))
	for _, team := range teams {
		members, err := r.listTeamMembers(ctx, team.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, teamToDomain(team, members))
	}
	return results, nil
}

func (r *Repo) GetTeamSpaceByPublicID(ctx context.Context, userID uint, publicID string) (*domaincollab.TeamSpace, error) {
	team, err := r.findTeamForMember(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	members, err := r.listTeamMembers(ctx, team.ID)
	if err != nil {
		return nil, err
	}
	result := teamToDomain(*team, members)
	return &result, nil
}

func (r *Repo) AddTeamMember(ctx context.Context, teamPublicID string, actorUserID uint, memberUserID uint, role string) (*domaincollab.TeamMember, error) {
	team, err := r.findTeamForOwner(ctx, actorUserID, teamPublicID)
	if err != nil {
		return nil, err
	}
	record := model.TeamMember{TeamID: team.ID, UserID: memberUserID, Role: role, InvitedBy: actorUserID}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "team_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "invited_by", "deleted_at"}),
	}).Create(&record).Error; err != nil {
		return nil, translateError(err)
	}
	var saved model.TeamMember
	if err := r.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", team.ID, memberUserID).First(&saved).Error; err != nil {
		return nil, translateError(err)
	}
	member := memberToDomain(saved, model.User{})
	return &member, nil
}

func (r *Repo) RemoveTeamMember(ctx context.Context, teamPublicID string, actorUserID uint, memberUserID uint) error {
	team, err := r.findTeamForOwner(ctx, actorUserID, teamPublicID)
	if err != nil {
		return err
	}
	if memberUserID == team.OwnerUserID {
		return repository.ErrInvalidInput
	}
	result := r.db.WithContext(ctx).Where("team_id = ? AND user_id = ?", team.ID, memberUserID).Delete(&model.TeamMember{})
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repo) FindUserByLogin(ctx context.Context, login string) (*domainuser.User, error) {
	normalized := strings.TrimSpace(login)
	if normalized == "" {
		return nil, repository.ErrInvalidInput
	}
	var user model.User
	query := r.db.WithContext(ctx).Where("username = ? OR email = ?", normalized, normalized)
	if err := query.First(&user).Error; err != nil {
		return nil, translateError(err)
	}
	result := domainuser.User{
		ID:          user.ID,
		PublicID:    user.PublicID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Email:       user.Email,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
	return &result, nil
}

func (r *Repo) findTeamForMember(ctx context.Context, userID uint, publicID string) (*model.TeamSpace, error) {
	var team model.TeamSpace
	if err := r.db.WithContext(ctx).
		Joins("JOIN team_members ON team_members.team_id = team_spaces.id AND team_members.deleted_at IS NULL").
		Where("team_spaces.public_id = ? AND team_members.user_id = ? AND team_spaces.status <> ?", strings.TrimSpace(publicID), userID, "deleted").
		First(&team).Error; err != nil {
		return nil, translateError(err)
	}
	return &team, nil
}

func (r *Repo) findTeamForOwner(ctx context.Context, userID uint, publicID string) (*model.TeamSpace, error) {
	var team model.TeamSpace
	if err := r.db.WithContext(ctx).
		Where("public_id = ? AND owner_user_id = ? AND status <> ?", strings.TrimSpace(publicID), userID, "deleted").
		First(&team).Error; err != nil {
		return nil, translateError(err)
	}
	return &team, nil
}

func (r *Repo) listTeamMembers(ctx context.Context, teamID uint) ([]domaincollab.TeamMember, error) {
	type row struct {
		model.TeamMember
		Username    string
		DisplayName string
		AvatarURL   string
		Email       string
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("team_members").
		Select("team_members.*, identity_users.username, identity_users.display_name, identity_users.avatar_url, identity_users.email").
		Joins("JOIN identity_users ON identity_users.id = team_members.user_id").
		Where("team_members.team_id = ? AND team_members.deleted_at IS NULL", teamID).
		Order("CASE team_members.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, team_members.created_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, translateError(err)
	}
	members := make([]domaincollab.TeamMember, 0, len(rows))
	for _, item := range rows {
		user := model.User{Username: item.Username, DisplayName: item.DisplayName, AvatarURL: item.AvatarURL, Email: item.Email}
		members = append(members, memberToDomain(item.TeamMember, user))
	}
	return members, nil
}

func scheduledPromptToRecord(item domaincollab.ScheduledPrompt) model.ScheduledPrompt {
	return model.ScheduledPrompt{
		PublicID:             item.PublicID,
		UserID:               item.UserID,
		AssistantID:          item.AssistantID,
		TargetConversationID: item.TargetConversationID,
		Title:                item.Title,
		Content:              item.Content,
		DueAt:                item.DueAt,
		NextRunAt:            item.NextRunAt,
		ScheduleType:         item.ScheduleType,
		ScheduleTime:         item.ScheduleTime,
		ScheduleWeekday:      item.ScheduleWeekday,
		CronExpression:       item.CronExpression,
		Model:                item.Model,
		Enabled:              item.Enabled,
		Status:               item.Status,
		LastTriggeredAt:      item.LastTriggeredAt,
		RetryCount:           item.RetryCount,
		LastError:            item.LastError,
		ConversationID:       item.ConversationID,
	}
}

func scheduledPromptToDomain(item model.ScheduledPrompt, conversationSlug string) domaincollab.ScheduledPrompt {
	return domaincollab.ScheduledPrompt{
		ID:                         item.ID,
		PublicID:                   item.PublicID,
		UserID:                     item.UserID,
		AssistantID:                item.AssistantID,
		TargetConversationID:       item.TargetConversationID,
		TargetConversationPublicID: conversationSlug,
		Title:                      item.Title,
		Content:                    item.Content,
		DueAt:                      item.DueAt,
		NextRunAt:                  item.NextRunAt,
		ScheduleType:               item.ScheduleType,
		ScheduleTime:               item.ScheduleTime,
		ScheduleWeekday:            item.ScheduleWeekday,
		CronExpression:             item.CronExpression,
		Model:                      item.Model,
		Enabled:                    item.Enabled,
		Status:                     item.Status,
		LastTriggeredAt:            item.LastTriggeredAt,
		RetryCount:                 item.RetryCount,
		LastError:                  item.LastError,
		ConversationID:             item.ConversationID,
		ConversationSlug:           conversationSlug,
		CreatedAt:                  item.CreatedAt,
		UpdatedAt:                  item.UpdatedAt,
	}
}

func teamToRecord(item domaincollab.TeamSpace) model.TeamSpace {
	return model.TeamSpace{PublicID: item.PublicID, OwnerUserID: item.OwnerUserID, Name: item.Name, Description: item.Description, Status: item.Status}
}

func teamToDomain(item model.TeamSpace, members []domaincollab.TeamMember) domaincollab.TeamSpace {
	return domaincollab.TeamSpace{
		ID:          item.ID,
		PublicID:    item.PublicID,
		OwnerUserID: item.OwnerUserID,
		Name:        item.Name,
		Description: item.Description,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		Members:     members,
	}
}

func memberToDomain(item model.TeamMember, user model.User) domaincollab.TeamMember {
	return domaincollab.TeamMember{
		ID:          item.ID,
		TeamID:      item.TeamID,
		UserID:      item.UserID,
		Role:        item.Role,
		InvitedBy:   item.InvitedBy,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Email:       user.Email,
	}
}
