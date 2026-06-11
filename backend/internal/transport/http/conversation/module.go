package conversation

import "context"

// Module 聚合会话 HTTP 处理器。
type Module struct {
	Handler *Handler
}

// NewModule 创建会话 HTTP 模块。
func NewModule(handler *Handler) *Module {
	return &Module{Handler: handler}
}

// GetPublicShareMetadata exposes share metadata to the frontend static HTML shim.
func (m *Module) GetPublicShareMetadata(ctx context.Context, shareID string) (string, string, error) {
	if m == nil || m.Handler == nil || m.Handler.service == nil {
		return "", "", nil
	}
	return m.Handler.service.GetPublicShareMetadata(ctx, shareID)
}
