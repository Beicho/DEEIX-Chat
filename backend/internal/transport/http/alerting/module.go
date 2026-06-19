package alerting

import "github.com/gin-gonic/gin"

// Module 封装告警管理 HTTP 模块。
type Module struct {
	handler *Handler
}

// NewModule 创建告警模块。
func NewModule(handler *Handler) *Module {
	return &Module{handler: handler}
}

// RegisterAdminRoutes 注册管理员侧告警配置与测试路由。
func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	group := adminGroup.Group("/alerting")
	{
		group.GET("/config", m.handler.GetConfig)
		group.PATCH("/config", m.handler.UpdateConfig)
		group.POST("/test-telegram", m.handler.TestTelegram)
		group.POST("/test-webhook", m.handler.TestWebhook)
	}
}
