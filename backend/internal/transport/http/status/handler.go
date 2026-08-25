package status

import (
	"context"
	"net/http"
	"time"

	domainstatus "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/status"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// Handler 处理公开状态页请求。
type Handler struct {
	service modelStatusService
}

type modelStatusService interface {
	GetModelsStatus(ctx context.Context) (*domainstatus.ModelsStatus, error)
}

// NewHandler 创建状态页 handler。
func NewHandler(service modelStatusService) *Handler {
	return &Handler{service: service}
}

// GetModelsStatus 返回模型可用性状态（公开端点）。
// @Summary 查询模型可用性状态
// @Description 返回所有模型的可用性状态（基于最近 24 小时调用数据）
// @Tags Status
// @Produce json
// @Success 200 {object} ModelsStatusResponse
// @Router /api/v1/status/models [get]
func (h *Handler) GetModelsStatus(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.service.GetModelsStatus(ctx)
	if err != nil {
		response.ErrorWithCode(c, http.StatusInternalServerError, response.CodeInternal, "failed to get models status")
		return
	}

	// 设置公开缓存头
	c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=120")
	c.JSON(http.StatusOK, toModelsStatusResponse(status))
}

// ModelsStatusResponse 表示模型状态响应。
type ModelsStatusResponse struct {
	OverallStatus string              `json:"overallStatus"` // "operational" | "degraded" | "down"
	LastUpdated   time.Time           `json:"lastUpdated"`
	Models        []ModelStatusDetail `json:"models"`
}

// ModelStatusDetail 表示单个模型的状态。
type ModelStatusDetail struct {
	ModelName    string    `json:"modelName"`
	Availability float64   `json:"availability"` // 0.0-1.0
	Status       string    `json:"status"`       // "operational" | "degraded" | "down"
	LastChecked  time.Time `json:"lastChecked"`
}

func toModelsStatusResponse(value *domainstatus.ModelsStatus) ModelsStatusResponse {
	if value == nil {
		return ModelsStatusResponse{Models: []ModelStatusDetail{}}
	}
	models := make([]ModelStatusDetail, 0, len(value.Models))
	for _, item := range value.Models {
		models = append(models, ModelStatusDetail{
			ModelName: item.ModelName, Availability: item.Availability, Status: item.Status, LastChecked: item.LastChecked,
		})
	}
	return ModelsStatusResponse{OverallStatus: value.OverallStatus, LastUpdated: value.LastUpdated, Models: models}
}
