package status

import "github.com/gin-gonic/gin"

// Module 封装状态页模块。
type Module struct {
	handler *Handler
}

// NewModule 创建状态页模块。
func NewModule(handler *Handler) *Module {
	return &Module{handler: handler}
}

// RegisterPublicRoutes 注册公开路由（无需鉴权）。
func (m *Module) RegisterPublicRoutes(router *gin.RouterGroup) {
	status := router.Group("/status")
	{
		status.GET("/models", m.handler.GetModelsStatus)
	}
}
