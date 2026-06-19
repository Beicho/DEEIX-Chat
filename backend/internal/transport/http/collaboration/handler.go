package collaboration

import (
	"errors"
	"net/http"
	"strconv"

	appcollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/collaboration"
	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// Handler exposes collaboration endpoints.
type Handler struct {
	service *appcollab.Service
}

// NewHandler creates a collaboration handler.
func NewHandler(service *appcollab.Service) *Handler {
	return &Handler{service: service}
}

func pageParams(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (h *Handler) ListAssistants(c *gin.Context) {
	page, pageSize := pageParams(c)
	items, total, err := h.service.ListAssistants(c.Request.Context(), middleware.MustUserID(c), page, pageSize, c.Query("includePublic") == "true")
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessPage(c, total, assistantResponses(items))
}

func (h *Handler) CreateAssistant(c *gin.Context) {
	var req AssistantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.CreateAssistant(c.Request.Context(), middleware.MustUserID(c), assistantInput(req))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, assistantResponse(*item))
}

func (h *Handler) UpdateAssistant(c *gin.Context) {
	var req AssistantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.UpdateAssistant(c.Request.Context(), middleware.MustUserID(c), c.Param("id"), assistantInput(req))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, assistantResponse(*item))
}

func (h *Handler) DeleteAssistant(c *gin.Context) {
	if err := h.service.DeleteAssistant(c.Request.Context(), middleware.MustUserID(c), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) ListMarketplaceAssistants(c *gin.Context) {
	page, pageSize := pageParams(c)
	items, total, err := h.service.ListPublicAssistants(c.Request.Context(), middleware.MustUserID(c), page, pageSize)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessPage(c, total, assistantResponses(items))
}

func (h *Handler) InstallAssistant(c *gin.Context) {
	if err := h.service.InstallAssistant(c.Request.Context(), middleware.MustUserID(c), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"installed": true})
}

func (h *Handler) UninstallAssistant(c *gin.Context) {
	if err := h.service.UninstallAssistant(c.Request.Context(), middleware.MustUserID(c), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"installed": false})
}

func (h *Handler) ListScheduledPrompts(c *gin.Context) {
	page, pageSize := pageParams(c)
	items, total, err := h.service.ListScheduledPrompts(c.Request.Context(), middleware.MustUserID(c), page, pageSize)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessPage(c, total, scheduledPromptResponses(items))
}

func (h *Handler) CreateScheduledPrompt(c *gin.Context) {
	var req ScheduledPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.CreateScheduledPrompt(c.Request.Context(), middleware.MustUserID(c), scheduledPromptInput(req))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, scheduledPromptResponse(*item))
}

func (h *Handler) UpdateScheduledPrompt(c *gin.Context) {
	var req ScheduledPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.UpdateScheduledPrompt(c.Request.Context(), middleware.MustUserID(c), c.Param("id"), scheduledPromptInput(req))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, scheduledPromptResponse(*item))
}

func (h *Handler) DeleteScheduledPrompt(c *gin.Context) {
	if err := h.service.DeleteScheduledPrompt(c.Request.Context(), middleware.MustUserID(c), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) ListTeamSpaces(c *gin.Context) {
	items, err := h.service.ListTeamSpaces(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, teamSpaceResponses(items))
}

func (h *Handler) CreateTeamSpace(c *gin.Context) {
	var req TeamSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.CreateTeamSpace(c.Request.Context(), middleware.MustUserID(c), appcollab.TeamSpaceInput{Name: req.Name, Description: req.Description})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, teamSpaceResponse(*item))
}

func (h *Handler) AddTeamMember(c *gin.Context) {
	var req TeamMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.AddTeamMember(c.Request.Context(), middleware.MustUserID(c), c.Param("id"), req.Login, req.Role)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, teamMemberResponse(*item))
}

func (h *Handler) RemoveTeamMember(c *gin.Context) {
	memberID, err := appcollab.ParseUserID(c.Param("user_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	if err := h.service.RemoveTeamMember(c.Request.Context(), middleware.MustUserID(c), c.Param("id"), memberID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appcollab.ErrAssistantNotFound), errors.Is(err, appcollab.ErrScheduledPromptNotFound), errors.Is(err, appcollab.ErrTeamSpaceNotFound), errors.Is(err, appcollab.ErrTeamMemberNotFound), errors.Is(err, appcollab.ErrUserNotFound):
		response.ErrorFrom(c, http.StatusNotFound, err)
	case errors.Is(err, appcollab.ErrInvalidInput):
		response.ErrorFrom(c, http.StatusBadRequest, err)
	case errors.Is(err, appcollab.ErrContentBlocked):
		response.ErrorWithCode(c, http.StatusBadRequest, "moderation_blocked", "message was blocked")
	default:
		response.Error(c, http.StatusInternalServerError, "collaboration operation failed")
	}
}

func assistantInput(req AssistantRequest) appcollab.AssistantInput {
	return appcollab.AssistantInput{
		Name:           req.Name,
		AvatarURL:      req.AvatarURL,
		Description:    req.Description,
		SystemPrompt:   req.SystemPrompt,
		DefaultModel:   req.DefaultModel,
		OpeningMessage: req.OpeningMessage,
		Visibility:     req.Visibility,
	}
}

func scheduledPromptInput(req ScheduledPromptRequest) appcollab.ScheduledPromptInput {
	return appcollab.ScheduledPromptInput{
		AssistantPublicID:          req.AssistantID,
		TargetConversationPublicID: req.TargetConversationID,
		Title:                      req.Title,
		Content:                    req.Content,
		DueAt:                      req.DueAt,
		ScheduleType:               req.ScheduleType,
		ScheduleTime:               req.ScheduleTime,
		ScheduleWeekday:            req.ScheduleWeekday,
		CronExpression:             req.CronExpression,
		Model:                      req.Model,
		Enabled:                    req.Enabled,
	}
}

func assistantResponses(items []domaincollab.Assistant) []AssistantResponse {
	results := make([]AssistantResponse, 0, len(items))
	for _, item := range items {
		results = append(results, assistantResponse(item))
	}
	return results
}

func assistantResponse(item domaincollab.Assistant) AssistantResponse {
	return AssistantResponse{
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
		Installed:      item.Installed,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func scheduledPromptResponses(items []domaincollab.ScheduledPrompt) []ScheduledPromptResponse {
	results := make([]ScheduledPromptResponse, 0, len(items))
	for _, item := range items {
		results = append(results, scheduledPromptResponse(item))
	}
	return results
}

func scheduledPromptResponse(item domaincollab.ScheduledPrompt) ScheduledPromptResponse {
	return ScheduledPromptResponse{
		PublicID:                item.PublicID,
		TargetConversationID:    item.TargetConversationPublicID,
		TargetConversationTitle: item.TargetConversationTitle,
		Title:                   item.Title,
		Content:                 item.Content,
		DueAt:                   item.DueAt,
		NextRunAt:               item.NextRunAt,
		ScheduleType:            item.ScheduleType,
		ScheduleTime:            item.ScheduleTime,
		ScheduleWeekday:         item.ScheduleWeekday,
		CronExpression:          item.CronExpression,
		Model:                   item.Model,
		Enabled:                 item.Enabled,
		Status:                  item.Status,
		LastTriggeredAt:         item.LastTriggeredAt,
		RetryCount:              item.RetryCount,
		LastError:               item.LastError,
		CreatedAt:               item.CreatedAt,
		UpdatedAt:               item.UpdatedAt,
	}
}

func teamSpaceResponses(items []domaincollab.TeamSpace) []TeamSpaceResponse {
	results := make([]TeamSpaceResponse, 0, len(items))
	for _, item := range items {
		results = append(results, teamSpaceResponse(item))
	}
	return results
}

func teamSpaceResponse(item domaincollab.TeamSpace) TeamSpaceResponse {
	return TeamSpaceResponse{
		PublicID:    item.PublicID,
		OwnerUserID: item.OwnerUserID,
		Name:        item.Name,
		Description: item.Description,
		Status:      item.Status,
		Members:     teamMemberResponses(item.Members),
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func teamMemberResponses(items []domaincollab.TeamMember) []TeamMemberResponse {
	results := make([]TeamMemberResponse, 0, len(items))
	for _, item := range items {
		results = append(results, teamMemberResponse(item))
	}
	return results
}

func teamMemberResponse(item domaincollab.TeamMember) TeamMemberResponse {
	return TeamMemberResponse{
		UserID:      item.UserID,
		Role:        item.Role,
		Username:    item.Username,
		DisplayName: item.DisplayName,
		AvatarURL:   item.AvatarURL,
		Email:       item.Email,
		CreatedAt:   item.CreatedAt,
	}
}
