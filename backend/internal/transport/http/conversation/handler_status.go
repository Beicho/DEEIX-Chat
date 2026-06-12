package conversation

import (
	"net/http"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

type ModelAvailabilityItemResponse struct {
	ModelName   string  `json:"modelName"`
	CallCount   int64   `json:"callCount"`
	SuccessRate float64 `json:"successRate"`
	Status      string  `json:"status"`
}

type ModelAvailabilityResponse struct {
	WindowHours int                             `json:"windowHours"`
	GeneratedAt string                          `json:"generatedAt"`
	Models      []ModelAvailabilityItemResponse `json:"models"`
}

func (h *Handler) GetModelAvailability(c *gin.Context) {
	items, err := h.service.ListModelAvailability(c.Request.Context(), 24*time.Hour)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load model status failed")
		return
	}
	results := make([]ModelAvailabilityItemResponse, 0, len(items))
	for _, item := range items {
		results = append(results, ModelAvailabilityItemResponse{
			ModelName:   item.ModelName,
			CallCount:   item.CallCount,
			SuccessRate: item.SuccessRate,
			Status:      item.Status,
		})
	}
	response.Success(c, ModelAvailabilityResponse{
		WindowHours: 24,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Models:      results,
	})
}
