package collaboration

// Module aggregates collaboration HTTP handlers.
type Module struct {
	Handler *Handler
}

// NewModule creates a collaboration module.
func NewModule(handler *Handler) *Module {
	return &Module{Handler: handler}
}
