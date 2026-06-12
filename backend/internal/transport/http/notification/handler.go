package notification

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// Handler 封装通知 HTTP 处理。
type Handler struct {
	service *appnotification.Service
}

// NewHandler 创建通知处理器。
func NewHandler(service *appnotification.Service) *Handler {
	return &Handler{service: service}
}

// ListNotifications godoc
// @Summary 查询通知列表
// @Description 登录用户分页查询站内通知；公告会作为虚拟通知合入
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param unread query bool false "仅未读"
// @Success 200 {object} NotificationListResponseDoc
// @Failure 400 {object} ErrorDoc
// @Failure 500 {object} ErrorDoc
// @Router /notifications [get]
func (h *Handler) ListNotifications(c *gin.Context) {
	page, pageSize := pageParams(c)
	unreadOnly, _ := strconv.ParseBool(c.Query("unread"))
	items, total, err := h.service.List(c.Request.Context(), middleware.MustUserID(c), appnotification.ListInput{
		Page:       page,
		PageSize:   pageSize,
		UnreadOnly: unreadOnly,
	}, time.Now())
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	response.SuccessPage(c, total, toNotificationResponses(items))
}

// UnreadCount godoc
// @Summary 查询未读通知数
// @Description 登录用户查询站内未读通知数
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UnreadCountResponseDoc
// @Failure 400 {object} ErrorDoc
// @Failure 500 {object} ErrorDoc
// @Router /notifications/unread-count [get]
func (h *Handler) UnreadCount(c *gin.Context) {
	count, err := h.service.UnreadCount(c.Request.Context(), middleware.MustUserID(c), time.Now())
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	response.Success(c, UnreadCountDataResponse{UnreadCount: count})
}

// MarkRead godoc
// @Summary 标记通知已读
// @Description 登录用户标记单条通知已读
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "通知ID"
// @Success 200 {object} NotificationReadResponseDoc
// @Failure 400 {object} ErrorDoc
// @Failure 404 {object} ErrorDoc
// @Failure 500 {object} ErrorDoc
// @Router /notifications/{id}/read [post]
func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), middleware.MustUserID(c), c.Param("id"), time.Now()); err != nil {
		writeNotificationError(c, err)
		return
	}
	response.Success(c, NotificationReadDataResponse{Read: true})
}

// MarkAllRead godoc
// @Summary 标记全部通知已读
// @Description 登录用户标记全部站内通知已读
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} NotificationReadResponseDoc
// @Failure 400 {object} ErrorDoc
// @Failure 500 {object} ErrorDoc
// @Router /notifications/read-all [post]
func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), middleware.MustUserID(c), time.Now()); err != nil {
		writeNotificationError(c, err)
		return
	}
	response.Success(c, NotificationReadDataResponse{Read: true})
}

func writeNotificationError(c *gin.Context, err error) {
	if errors.Is(err, appnotification.ErrNotificationNotFound) {
		response.Error(c, http.StatusNotFound, "notification not found")
		return
	}
	if errors.Is(err, appnotification.ErrInvalidNotification) {
		response.ErrorFrom(c, http.StatusBadRequest, err)
		return
	}
	response.Error(c, http.StatusInternalServerError, "notification operation failed")
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
