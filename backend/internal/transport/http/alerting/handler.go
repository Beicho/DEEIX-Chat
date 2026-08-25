package alerting

import (
	"errors"
	"net/http"

	appalerting "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/alerting"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// Handler 封装告警配置与测试发送的管理端 HTTP 处理。
type Handler struct {
	service *appalerting.Service
}

type configResponse struct {
	Enabled            bool     `json:"enabled"`
	EnabledNotifiers   []string `json:"enabledNotifiers"`
	TelegramConfigured bool     `json:"telegramConfigured"`
	TelegramChatID     string   `json:"telegramChatId"`
	WebhookConfigured  bool     `json:"webhookConfigured"`
	DebounceSeconds    int      `json:"debounceSeconds"`
}

func toConfigResponse(view appalerting.ConfigView) configResponse {
	return configResponse{
		Enabled:            view.Enabled,
		EnabledNotifiers:   view.EnabledNotifiers,
		TelegramConfigured: view.TelegramConfigured,
		TelegramChatID:     view.TelegramChatID,
		WebhookConfigured:  view.WebhookConfigured,
		DebounceSeconds:    view.DebounceSeconds,
	}
}

// NewHandler 创建处理器。
func NewHandler(service *appalerting.Service) *Handler {
	return &Handler{service: service}
}

// GetConfig godoc
// @Summary 查询告警配置
// @Tags admin/alerting
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /admin/alerting/config [get]
func (h *Handler) GetConfig(c *gin.Context) {
	view, err := h.service.GetConfigView(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "load alerting config failed")
		return
	}
	response.Success(c, toConfigResponse(view))
}

// UpdateConfig godoc
// @Summary 更新告警配置
// @Tags admin/alerting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /admin/alerting/config [patch]
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidRequestBody(c, err)
		return
	}
	view, err := h.service.UpdateConfig(c.Request.Context(), req.toInput())
	if err != nil {
		if errors.Is(err, appalerting.ErrInvalidConfig) {
			response.ErrorFrom(c, http.StatusBadRequest, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "update alerting config failed")
		return
	}
	response.Success(c, toConfigResponse(view))
}

// TestTelegram godoc
// @Summary 测试 Telegram 告警
// @Tags admin/alerting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /admin/alerting/test-telegram [post]
func (h *Handler) TestTelegram(c *gin.Context) {
	var req TestTelegramRequest
	_ = bindOptionalJSON(c, &req)
	err := h.service.SendTest(c.Request.Context(), appalerting.NotifierTelegram, appalerting.TestOverride{
		TelegramBotToken: req.BotToken,
		TelegramChatID:   req.ChatID,
	})
	h.respondTest(c, err)
}

// TestWebhook godoc
// @Summary 测试 Webhook 告警
// @Tags admin/alerting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /admin/alerting/test-webhook [post]
func (h *Handler) TestWebhook(c *gin.Context) {
	var req TestWebhookRequest
	_ = bindOptionalJSON(c, &req)
	err := h.service.SendTest(c.Request.Context(), appalerting.NotifierWebhook, appalerting.TestOverride{
		WebhookURL: req.URL,
	})
	h.respondTest(c, err)
}

func (h *Handler) respondTest(c *gin.Context, err error) {
	if err != nil {
		if errors.Is(err, appalerting.ErrInvalidConfig) {
			response.ErrorFrom(c, http.StatusBadRequest, err)
			return
		}
		// 发送失败属于上游/网络问题，按 502 处理，且不回显敏感细节。
		response.Error(c, http.StatusBadGateway, "alert delivery failed")
		return
	}
	response.Success(c, gin.H{"delivered": true})
}

// bindOptionalJSON 在请求体存在时解析，空请求体时不报错。
func bindOptionalJSON(c *gin.Context, target interface{}) error {
	if c.Request == nil || c.Request.ContentLength == 0 {
		return nil
	}
	return c.ShouldBindJSON(target)
}
