package notification

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册通知用户侧路由。
func (m *Module) RegisterRoutes(authRequired *gin.RouterGroup) {
	authRequired.GET("/notifications", m.Handler.ListNotifications)
	authRequired.GET("/notifications/unread-count", m.Handler.UnreadCount)
	authRequired.POST("/notifications/read-all", m.Handler.MarkAllRead)
	authRequired.POST("/notifications/:id/read", m.Handler.MarkRead)
}
