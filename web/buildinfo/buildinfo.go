package buildinfo

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

const DefaultPath = "/api"

type Option func(*Handler)

type Handler struct {
	defaultPath string
}

func Use(opts ...Option) di.Node {
	return di.Provide(func() *Handler {
		return NewHandler(opts...)
	})
}

func WithDefaultPath(path ...string) Option {
	return func(h *Handler) {
		if len(path) == 0 {
			h.defaultPath = DefaultPath
			return
		}
		h.defaultPath = path[0]
	}
}
