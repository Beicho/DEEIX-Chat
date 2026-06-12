package collaboration

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Service owns assistant presets, scheduled prompts, and team spaces.
type Service struct {
	repo          repository.CollaborationRepository
	notifications *appnotification.Service
	conversations scheduledPromptConversationService
	logger        *zap.Logger
}

type scheduledPromptConversationService interface {
	CreateConversation(ctx context.Context, userID uint, title string, modelName string, projectPublicID string) (*domainconversation.Conversation, error)
	SendScheduledPromptMessage(ctx context.Context, input ScheduledPromptSendInput) (*ScheduledPromptSendResult, error)
}

type ScheduledPromptSendInput struct {
	UserID            uint
	ConversationID    uint
	RequestID         string
	Content           string
	PlatformModelName string
	AssistantPublicID string
}

type ScheduledPromptSendResult struct {
	AssistantContent string
}

// NewService creates the collaboration service.
func NewService(repo repository.CollaborationRepository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// SetNotificationService enables due scheduled prompts to create reminders.
func (s *Service) SetNotificationService(notifications *appnotification.Service) {
	s.notifications = notifications
}

// SetConversationService enables due scheduled prompts to write results into conversations.
func (s *Service) SetConversationService(conversations scheduledPromptConversationService) {
	s.conversations = conversations
}

type AssistantInput struct {
	Name           string
	AvatarURL      string
	Description    string
	SystemPrompt   string
	DefaultModel   string
	OpeningMessage string
	Visibility     string
}

type ScheduledPromptInput struct {
	AssistantPublicID          string
	TargetConversationPublicID string
	Title                      string
	Content                    string
	DueAt                      time.Time
	ScheduleType               string
	ScheduleTime               string
	ScheduleWeekday            int
	CronExpression             string
	Model                      string
	Enabled                    bool
}

type TeamSpaceInput struct {
	Name        string
	Description string
}

func (s *Service) CreateAssistant(ctx context.Context, userID uint, input AssistantInput) (*domaincollab.Assistant, error) {
	item := &domaincollab.Assistant{
		PublicID:       normalizePublicID(uuid.NewString()),
		OwnerUserID:    userID,
		Name:           clamp(input.Name, 80),
		AvatarURL:      clamp(input.AvatarURL, 2048),
		Description:    clamp(input.Description, 255),
		SystemPrompt:   clamp(input.SystemPrompt, 12000),
		DefaultModel:   clamp(input.DefaultModel, 128),
		OpeningMessage: clamp(input.OpeningMessage, 2000),
		Visibility:     normalizeVisibility(input.Visibility),
		Status:         "active",
	}
	if item.Name == "" || item.SystemPrompt == "" || userID == 0 {
		return nil, ErrInvalidInput
	}
	if item.Visibility == domaincollab.AssistantVisibilityPublic {
		now := time.Now()
		item.PublishedAt = &now
	}
	if err := s.repo.CreateAssistant(ctx, item); err != nil {
		return nil, mapRepoError(err, ErrAssistantNotFound)
	}
	return item, nil
}

func (s *Service) ListAssistants(ctx context.Context, userID uint, page int, pageSize int, includePublic bool) ([]domaincollab.Assistant, int64, error) {
	offset, limit := normalizePage(page, pageSize)
	return s.repo.ListAssistants(ctx, userID, includePublic, offset, limit)
}

func (s *Service) ListPublicAssistants(ctx context.Context, userID uint, page int, pageSize int) ([]domaincollab.Assistant, int64, error) {
	offset, limit := normalizePage(page, pageSize)
	return s.repo.ListPublicAssistants(ctx, userID, offset, limit)
}

func (s *Service) UpdateAssistant(ctx context.Context, userID uint, publicID string, input AssistantInput) (*domaincollab.Assistant, error) {
	visibility := normalizeVisibility(input.Visibility)
	patch := repository.AssistantPatch{
		Name:           stringPtr(clamp(input.Name, 80)),
		AvatarURL:      stringPtr(clamp(input.AvatarURL, 2048)),
		Description:    stringPtr(clamp(input.Description, 255)),
		SystemPrompt:   stringPtr(clamp(input.SystemPrompt, 12000)),
		DefaultModel:   stringPtr(clamp(input.DefaultModel, 128)),
		OpeningMessage: stringPtr(clamp(input.OpeningMessage, 2000)),
		Visibility:     &visibility,
	}
	if visibility == domaincollab.AssistantVisibilityPublic {
		now := time.Now()
		patch.PublishedAt = &now
	}
	item, err := s.repo.UpdateAssistant(ctx, userID, publicID, patch)
	return item, mapRepoError(err, ErrAssistantNotFound)
}

func (s *Service) DeleteAssistant(ctx context.Context, userID uint, publicID string) error {
	return mapRepoError(s.repo.DeleteAssistant(ctx, userID, publicID), ErrAssistantNotFound)
}

func (s *Service) InstallAssistant(ctx context.Context, userID uint, publicID string) error {
	_, err := s.repo.InstallAssistant(ctx, userID, publicID)
	return mapRepoError(err, ErrAssistantNotFound)
}

func (s *Service) UninstallAssistant(ctx context.Context, userID uint, publicID string) error {
	return mapRepoError(s.repo.UninstallAssistant(ctx, userID, publicID), ErrAssistantNotFound)
}

func (s *Service) ResolveAssistantPrompt(ctx context.Context, userID uint, publicID string) (*domaincollab.Assistant, error) {
	item, err := s.repo.GetInstalledAssistantPrompt(ctx, userID, publicID)
	return item, mapRepoError(err, ErrAssistantNotFound)
}

func (s *Service) CreateScheduledPrompt(ctx context.Context, userID uint, input ScheduledPromptInput) (*domaincollab.ScheduledPrompt, error) {
	if userID == 0 {
		return nil, ErrInvalidInput
	}
	var assistantID *uint
	if strings.TrimSpace(input.AssistantPublicID) != "" {
		assistant, err := s.ResolveAssistantPrompt(ctx, userID, input.AssistantPublicID)
		if err != nil {
			return nil, err
		}
		assistantID = &assistant.ID
	}
	var targetConversationID *uint
	targetConversationPublicID := strings.TrimSpace(input.TargetConversationPublicID)
	if targetConversationPublicID != "" {
		conversationID, _, err := s.repo.GetConversationTarget(ctx, userID, targetConversationPublicID)
		if err != nil {
			return nil, mapRepoError(err, ErrScheduledPromptNotFound)
		}
		targetConversationID = &conversationID
	}
	now := time.Now()
	scheduleType := normalizeScheduleType(input.ScheduleType)
	nextRunAt := input.DueAt
	if scheduleType == scheduleTypeOnce {
		if nextRunAt.IsZero() {
			return nil, ErrInvalidInput
		}
	} else {
		computed, err := computeNextRunAt(scheduleType, input.ScheduleTime, input.ScheduleWeekday, input.CronExpression, now)
		if err != nil {
			return nil, ErrInvalidInput
		}
		nextRunAt = computed
	}
	item := &domaincollab.ScheduledPrompt{
		PublicID:                   normalizePublicID(uuid.NewString()),
		UserID:                     userID,
		AssistantID:                assistantID,
		TargetConversationID:       targetConversationID,
		TargetConversationPublicID: targetConversationPublicID,
		Title:                      clamp(firstNonEmpty(input.Title, input.Content), 120),
		Content:                    clamp(input.Content, 20000),
		DueAt:                      nextRunAt,
		NextRunAt:                  nextRunAt,
		ScheduleType:               scheduleType,
		ScheduleTime:               clamp(input.ScheduleTime, 8),
		ScheduleWeekday:            input.ScheduleWeekday,
		CronExpression:             clamp(input.CronExpression, 128),
		Model:                      clamp(input.Model, 128),
		Enabled:                    input.Enabled,
		Status:                     "scheduled",
	}
	if item.Title == "" || item.Content == "" {
		return nil, ErrInvalidInput
	}
	if err := s.repo.CreateScheduledPrompt(ctx, item); err != nil {
		return nil, mapRepoError(err, ErrScheduledPromptNotFound)
	}
	return item, nil
}

func (s *Service) ListScheduledPrompts(ctx context.Context, userID uint, page int, pageSize int) ([]domaincollab.ScheduledPrompt, int64, error) {
	offset, limit := normalizePage(page, pageSize)
	return s.repo.ListScheduledPrompts(ctx, userID, offset, limit)
}

func (s *Service) UpdateScheduledPrompt(ctx context.Context, userID uint, publicID string, input ScheduledPromptInput) (*domaincollab.ScheduledPrompt, error) {
	var assistantID **uint
	if strings.TrimSpace(input.AssistantPublicID) != "" {
		assistant, err := s.ResolveAssistantPrompt(ctx, userID, input.AssistantPublicID)
		if err != nil {
			return nil, err
		}
		resolvedAssistantID := &assistant.ID
		assistantID = &resolvedAssistantID
	}
	var targetConversationID **uint
	if strings.TrimSpace(input.TargetConversationPublicID) != "" {
		conversationID, _, err := s.repo.GetConversationTarget(ctx, userID, input.TargetConversationPublicID)
		if err != nil {
			return nil, mapRepoError(err, ErrScheduledPromptNotFound)
		}
		resolvedTargetID := &conversationID
		targetConversationID = &resolvedTargetID
	}
	now := time.Now()
	scheduleType := normalizeScheduleType(input.ScheduleType)
	nextRunAt := input.DueAt
	if scheduleType != scheduleTypeOnce {
		computed, err := computeNextRunAt(scheduleType, input.ScheduleTime, input.ScheduleWeekday, input.CronExpression, now)
		if err != nil {
			return nil, ErrInvalidInput
		}
		nextRunAt = computed
	}
	if nextRunAt.IsZero() {
		return nil, ErrInvalidInput
	}
	enabled := input.Enabled
	retryCount := 0
	lastError := ""
	patch := repository.ScheduledPromptPatch{
		AssistantID:          assistantID,
		TargetConversationID: targetConversationID,
		Title:                stringPtr(clamp(firstNonEmpty(input.Title, input.Content), 120)),
		Content:              stringPtr(clamp(input.Content, 20000)),
		DueAt:                &nextRunAt,
		NextRunAt:            &nextRunAt,
		ScheduleType:         stringPtr(scheduleType),
		ScheduleTime:         stringPtr(clamp(input.ScheduleTime, 8)),
		ScheduleWeekday:      &input.ScheduleWeekday,
		CronExpression:       stringPtr(clamp(input.CronExpression, 128)),
		Model:                stringPtr(clamp(input.Model, 128)),
		Enabled:              &enabled,
		Status:               stringPtr("scheduled"),
		RetryCount:           &retryCount,
		LastError:            &lastError,
	}
	item, err := s.repo.UpdateScheduledPrompt(ctx, userID, publicID, patch)
	return item, mapRepoError(err, ErrScheduledPromptNotFound)
}

func (s *Service) DeleteScheduledPrompt(ctx context.Context, userID uint, publicID string) error {
	return mapRepoError(s.repo.DeleteScheduledPrompt(ctx, userID, publicID), ErrScheduledPromptNotFound)
}

func (s *Service) CreateTeamSpace(ctx context.Context, userID uint, input TeamSpaceInput) (*domaincollab.TeamSpace, error) {
	team := &domaincollab.TeamSpace{
		PublicID:    normalizePublicID(uuid.NewString()),
		OwnerUserID: userID,
		Name:        clamp(input.Name, 80),
		Description: clamp(input.Description, 255),
		Status:      "active",
	}
	if userID == 0 || team.Name == "" {
		return nil, ErrInvalidInput
	}
	owner := &domaincollab.TeamMember{UserID: userID, Role: domaincollab.TeamRoleOwner, InvitedBy: userID}
	if err := s.repo.CreateTeamSpace(ctx, team, owner); err != nil {
		return nil, mapRepoError(err, ErrTeamSpaceNotFound)
	}
	return team, nil
}

func (s *Service) ListTeamSpaces(ctx context.Context, userID uint) ([]domaincollab.TeamSpace, error) {
	return s.repo.ListTeamSpaces(ctx, userID)
}

func (s *Service) AddTeamMember(ctx context.Context, actorUserID uint, teamID string, login string, role string) (*domaincollab.TeamMember, error) {
	user, err := s.repo.FindUserByLogin(ctx, login)
	if err != nil {
		return nil, mapRepoError(err, ErrUserNotFound)
	}
	normalizedRole := normalizeTeamRole(role)
	member, err := s.repo.AddTeamMember(ctx, teamID, actorUserID, user.ID, normalizedRole)
	return member, mapRepoError(err, ErrTeamSpaceNotFound)
}

func (s *Service) RemoveTeamMember(ctx context.Context, actorUserID uint, teamID string, memberUserID uint) error {
	return mapRepoError(s.repo.RemoveTeamMember(ctx, teamID, actorUserID, memberUserID), ErrTeamMemberNotFound)
}

// StartScheduledPromptWorker scans due prompts and writes model results into conversations.
func (s *Service) StartScheduledPromptWorker(ctx context.Context) {
	if s == nil || s.repo == nil || s.conversations == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		s.processDueScheduledPrompts(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.processDueScheduledPrompts(ctx)
			}
		}
	}()
}

func (s *Service) processDueScheduledPrompts(ctx context.Context) {
	now := time.Now()
	items, err := s.repo.ListDueScheduledPrompts(ctx, now, 50)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("scheduled_prompt_scan_failed", zap.Error(err))
		}
		return
	}
	for _, item := range items {
		if err := s.runScheduledPrompt(ctx, item, now); err != nil && s.logger != nil {
			s.logger.Warn("scheduled_prompt_run_failed", zap.Uint("scheduled_prompt_id", item.ID), zap.Error(err))
		}
	}
}

func (s *Service) runScheduledPrompt(ctx context.Context, item domaincollab.ScheduledPrompt, now time.Time) error {
	conversationID := uint(0)
	conversationPublicID := strings.TrimSpace(item.TargetConversationPublicID)
	if item.TargetConversationID != nil && *item.TargetConversationID > 0 {
		conversationID = *item.TargetConversationID
	}
	if conversationID == 0 && conversationPublicID != "" {
		resolvedID, _, err := s.repo.GetConversationTarget(ctx, item.UserID, conversationPublicID)
		if err != nil {
			return s.handleScheduledPromptFailure(ctx, item, now, err)
		}
		conversationID = resolvedID
	}
	if conversationID == 0 {
		conversation, err := s.conversations.CreateConversation(ctx, item.UserID, item.Title, item.Model, "")
		if err != nil {
			return s.handleScheduledPromptFailure(ctx, item, now, err)
		}
		conversationID = conversation.ID
		conversationPublicID = conversation.PublicID
	}

	result, err := s.conversations.SendScheduledPromptMessage(ctx, ScheduledPromptSendInput{
		UserID:            item.UserID,
		ConversationID:    conversationID,
		RequestID:         "scheduled_prompt_" + item.PublicID,
		Content:           item.Content,
		PlatformModelName: item.Model,
		AssistantPublicID: item.AssistantPublicID,
	})
	if err != nil {
		return s.handleScheduledPromptFailure(ctx, item, now, err)
	}
	nextRunAt, err := s.nextRunAfterSuccess(item, now)
	if err != nil {
		return s.handleScheduledPromptFailure(ctx, item, now, err)
	}
	if err := s.repo.MarkScheduledPromptSucceeded(ctx, item.ID, now, nextRunAt, conversationID); err != nil {
		return err
	}
	if s.notifications != nil {
		link := "/chat"
		if conversationPublicID != "" {
			link = "/chat/" + conversationPublicID
		}
		body := item.Content
		if result != nil {
			body = result.AssistantContent
		}
		_, _ = s.notifications.Create(ctx, item.UserID, appnotification.CreateInput{
			Type:     "scheduled_prompt",
			Title:    item.Title,
			Body:     body,
			Link:     link,
			Source:   "scheduled_prompt",
			SourceID: item.PublicID,
		})
	}
	return nil
}

func (s *Service) nextRunAfterSuccess(item domaincollab.ScheduledPrompt, now time.Time) (*time.Time, error) {
	if normalizeScheduleType(item.ScheduleType) == scheduleTypeOnce {
		return nil, nil
	}
	next, err := computeNextRunAt(item.ScheduleType, item.ScheduleTime, item.ScheduleWeekday, item.CronExpression, now)
	if err != nil {
		return nil, err
	}
	return &next, nil
}

func (s *Service) handleScheduledPromptFailure(ctx context.Context, item domaincollab.ScheduledPrompt, now time.Time, cause error) error {
	message := clamp(cause.Error(), 2000)
	disable := item.RetryCount >= 1
	var retryAt *time.Time
	if !disable {
		next := now.Add(time.Minute)
		retryAt = &next
	}
	if err := s.repo.MarkScheduledPromptFailed(ctx, item.ID, message, retryAt, disable); err != nil {
		return err
	}
	if disable && s.notifications != nil {
		_, _ = s.notifications.Create(ctx, item.UserID, appnotification.CreateInput{
			Type:     "scheduled_prompt",
			Title:    item.Title,
			Body:     message,
			Link:     "/assistants",
			Source:   "scheduled_prompt",
			SourceID: item.PublicID,
		})
	}
	return cause
}

func normalizePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return (page - 1) * pageSize, pageSize
}

func normalizeVisibility(value string) string {
	if strings.TrimSpace(value) == domaincollab.AssistantVisibilityPublic {
		return domaincollab.AssistantVisibilityPublic
	}
	return domaincollab.AssistantVisibilityPrivate
}

func normalizeTeamRole(value string) string {
	switch strings.TrimSpace(value) {
	case domaincollab.TeamRoleAdmin:
		return domaincollab.TeamRoleAdmin
	default:
		return domaincollab.TeamRoleMember
	}
}

func mapRepoError(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) {
		return notFound
	}
	if errors.Is(err, repository.ErrInvalidInput) {
		return ErrInvalidInput
	}
	return err
}

func normalizePublicID(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "-", "")
}

func clamp(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringPtr(value string) *string {
	return &value
}

func ParseUserID(value string) (uint, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed == 0 {
		return 0, ErrInvalidInput
	}
	return uint(parsed), nil
}
