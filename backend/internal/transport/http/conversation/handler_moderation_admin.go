package conversation

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ModerationEventResponse struct {
	ID             uint    `json:"id"`
	UserID         uint    `json:"userID"`
	ConversationID uint    `json:"conversationID"`
	MessageID      uint    `json:"messageID"`
	RunID          string  `json:"runID"`
	Direction      string  `json:"direction"`
	Action         string  `json:"action"`
	Model          string  `json:"model"`
	Score          float64 `json:"score"`
	Threshold      float64 `json:"threshold"`
	Flagged        bool    `json:"flagged"`
	CategoriesJSON string  `json:"categoriesJSON"`
	Reason         string  `json:"reason"`
	ReviewStatus   string  `json:"reviewStatus"`
	ReviewedBy     uint    `json:"reviewedBy"`
	ReviewedAt     *string `json:"reviewedAt"`
	ReviewNote     string  `json:"reviewNote"`
	Disposition    string  `json:"disposition"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

type ModerationReviewRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (h *Handler) ListModerationEvents(c *gin.Context) {
	page, pageSize := pageParams(c)
	items, total, err := h.service.ListModerationEvents(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list moderation events failed")
		return
	}
	results := make([]ModerationEventResponse, 0, len(items))
	for _, item := range items {
		results = append(results, moderationEventResponse(item))
	}
	response.SuccessPage(c, total, results)
}

func (h *Handler) UpdateModerationReview(c *gin.Context) {
	id, err := moderationEventIDParam(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid moderation event id")
		return
	}
	var req ModerationReviewRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.UpdateModerationReview(c.Request.Context(), id, middleware.MustUserID(c), req.Status, req.Note)
	if err != nil {
		writeModerationAdminError(c, err)
		return
	}
	response.Success(c, moderationEventResponse(*item))
}

func (h *Handler) ReleaseModerationDisposition(c *gin.Context) {
	id, err := moderationEventIDParam(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid moderation event id")
		return
	}
	var req ModerationReviewRequest
	_ = c.ShouldBindJSON(&req)
	item, err := h.service.ReleaseModerationDisposition(c.Request.Context(), id, middleware.MustUserID(c), req.Note)
	if err != nil {
		writeModerationAdminError(c, err)
		return
	}
	response.Success(c, moderationEventResponse(*item))
}

func moderationEventResponse(item model.ModerationEvent) ModerationEventResponse {
	var reviewedAt *string
	if item.ReviewedAt != nil {
		value := item.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		reviewedAt = &value
	}
	return ModerationEventResponse{
		ID:             item.ID,
		UserID:         item.UserID,
		ConversationID: item.ConversationID,
		MessageID:      item.MessageID,
		RunID:          item.RunID,
		Direction:      item.Direction,
		Action:         item.Action,
		Model:          item.Model,
		Score:          item.Score,
		Threshold:      item.Threshold,
		Flagged:        item.Flagged,
		CategoriesJSON: item.CategoriesJSON,
		Reason:         item.Reason,
		ReviewStatus:   item.ReviewStatus,
		ReviewedBy:     item.ReviewedBy,
		ReviewedAt:     reviewedAt,
		ReviewNote:     item.ReviewNote,
		Disposition:    item.Disposition,
		CreatedAt:      item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func moderationEventIDParam(c *gin.Context) (uint, error) {
	raw := strings.TrimSpace(c.Param("id"))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, err
	}
	return uint(value), nil
}

func writeModerationAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appconversation.ErrMessageNotFound):
		response.Error(c, http.StatusNotFound, "moderation event not found")
	case errors.Is(err, appconversation.ErrInvalidMessageContent):
		response.Error(c, http.StatusBadRequest, "invalid moderation review status")
	default:
		response.Error(c, http.StatusInternalServerError, "update moderation event failed")
	}
}
