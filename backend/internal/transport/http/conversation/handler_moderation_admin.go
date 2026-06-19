package conversation

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ModerationEventResponse struct {
	ID                    uint    `json:"id"`
	UserID                uint    `json:"userID"`
	ConversationID        uint    `json:"conversationID"`
	MessageID             uint    `json:"messageID"`
	RunID                 string  `json:"runID"`
	Direction             string  `json:"direction"`
	Action                string  `json:"action"`
	Model                 string  `json:"model"`
	Score                 float64 `json:"score"`
	Threshold             float64 `json:"threshold"`
	Flagged               bool    `json:"flagged"`
	CategoriesJSON        string  `json:"categoriesJSON"`
	Reason                string  `json:"reason"`
	ReviewStatus          string  `json:"reviewStatus"`
	ReviewedBy            uint    `json:"reviewedBy"`
	ReviewedAt            *string `json:"reviewedAt"`
	ReviewNote            string  `json:"reviewNote"`
	Disposition           string  `json:"disposition"`
	ContentSnapshot       string  `json:"contentSnapshot"`
	ContentHash           string  `json:"contentHash"`
	SnapshotTruncated     bool    `json:"snapshotTruncated"`
	EventType             string  `json:"eventType"`
	DispositionAppliedAt  *string `json:"dispositionAppliedAt"`
	DispositionReleasedAt *string `json:"dispositionReleasedAt"`
	DispositionReleasedBy uint    `json:"dispositionReleasedBy"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
}

type ModerationReviewRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (h *Handler) ListModerationEvents(c *gin.Context) {
	page, pageSize := pageParams(c)
	filter, err := moderationEventFilterFromQuery(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid moderation event filter")
		return
	}
	items, total, err := h.service.ListModerationEvents(c.Request.Context(), page, pageSize, filter)
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
	h.recordAudit(c, "update_moderation_review", "moderation_event", strconv.FormatUint(uint64(item.ID), 10), map[string]interface{}{
		"userID":      item.UserID,
		"status":      item.ReviewStatus,
		"reviewNote":  item.ReviewNote,
		"disposition": item.Disposition,
	})
	response.Success(c, moderationEventResponse(*item))
}

func (h *Handler) ReleaseModerationDisposition(c *gin.Context) {
	id, err := moderationEventIDParam(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid moderation event id")
		return
	}
	var req ModerationReviewRequest
	if err = c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.InvalidRequestBody(c, err)
		return
	}
	item, err := h.service.ReleaseModerationDisposition(c.Request.Context(), id, middleware.MustUserID(c), req.Note)
	if err != nil {
		writeModerationAdminError(c, err)
		return
	}
	h.recordAudit(c, "release_moderation_disposition", "moderation_event", strconv.FormatUint(uint64(item.ID), 10), map[string]interface{}{
		"userID":      item.UserID,
		"disposition": item.Disposition,
		"reviewNote":  item.ReviewNote,
	})
	response.Success(c, moderationEventResponse(*item))
}

func moderationEventResponse(item model.ModerationEvent) ModerationEventResponse {
	var reviewedAt *string
	if item.ReviewedAt != nil {
		value := item.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		reviewedAt = &value
	}
	var dispositionAppliedAt *string
	if item.DispositionAppliedAt != nil {
		value := item.DispositionAppliedAt.Format("2006-01-02T15:04:05Z07:00")
		dispositionAppliedAt = &value
	}
	var dispositionReleasedAt *string
	if item.DispositionReleasedAt != nil {
		value := item.DispositionReleasedAt.Format("2006-01-02T15:04:05Z07:00")
		dispositionReleasedAt = &value
	}
	return ModerationEventResponse{
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
		ReviewStatus:          item.ReviewStatus,
		ReviewedBy:            item.ReviewedBy,
		ReviewedAt:            reviewedAt,
		ReviewNote:            item.ReviewNote,
		Disposition:           item.Disposition,
		ContentSnapshot:       item.ContentSnapshot,
		ContentHash:           item.ContentHash,
		SnapshotTruncated:     item.SnapshotTruncated,
		EventType:             item.EventType,
		DispositionAppliedAt:  dispositionAppliedAt,
		DispositionReleasedAt: dispositionReleasedAt,
		DispositionReleasedBy: item.DispositionReleasedBy,
		CreatedAt:             item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

func moderationEventFilterFromQuery(c *gin.Context) (model.ModerationEventFilter, error) {
	var filter model.ModerationEventFilter
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return filter, err
		}
		filter.UserID = uint(value)
	}
	filter.Direction = strings.TrimSpace(c.Query("direction"))
	filter.ReviewStatus = strings.TrimSpace(c.Query("review_status"))
	filter.Disposition = strings.TrimSpace(c.Query("disposition"))
	filter.EventType = strings.TrimSpace(c.Query("event_type"))
	if raw := strings.TrimSpace(c.Query("flagged")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return filter, err
		}
		filter.Flagged = &value
	}
	if raw := strings.TrimSpace(c.Query("created_from")); raw != "" {
		value, err := parseModerationEventTime(raw, false)
		if err != nil {
			return filter, err
		}
		filter.CreatedFrom = &value
	}
	if raw := strings.TrimSpace(c.Query("created_to")); raw != "" {
		value, err := parseModerationEventTime(raw, true)
		if err != nil {
			return filter, err
		}
		filter.CreatedTo = &value
	}
	return filter, nil
}

func parseModerationEventTime(raw string, endOfDay bool) (time.Time, error) {
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		return value, nil
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return value.Add(24*time.Hour - time.Nanosecond), nil
	}
	return value, nil
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
