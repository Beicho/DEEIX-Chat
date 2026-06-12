package notification

// Module 聚合通知 HTTP 处理器。
type Module struct {
	Handler *Handler
}

// NewModule 创建通知 HTTP 模块。
func NewModule(handler *Handler) *Module {
	return &Module{Handler: handler}
}
