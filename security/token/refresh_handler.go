package token

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type RefreshHandler struct {
	service   Manager
	path      string
	deliverer PairDeliverer
	resolver  PairDelivererResolver
}

func NewRefreshHandler(service Manager) *RefreshHandler {
	return &RefreshHandler{
		service:   service,
		path:      "/api/v1/auth/refresh",
		deliverer: JSONPairDeliverer(),
	}
}

func (h *RefreshHandler) WithPath(path string) *RefreshHandler {
	if path != "" {
		h.path = path
	}
	return h
}

func (h *RefreshHandler) Handle(r web.Router) {
	r.Post(h.path, h.service.RefreshMiddleware(), h.Refresh)
}

func (h *RefreshHandler) WithDeliverer(deliverer PairDeliverer) *RefreshHandler {
	if deliverer != nil {
		h.deliverer = deliverer
		h.resolver = nil
	}
	return h
}

func (h *RefreshHandler) WithDelivererResolver(resolver PairDelivererResolver) *RefreshHandler {
	if resolver != nil {
		h.resolver = resolver
	}
	return h
}

func (h *RefreshHandler) Refresh(c fiber.Ctx) error {
	sub, err := SubjectFromContext(c)
	if err != nil {
		return unauthorized(c, err)
	}

	pair, err := h.service.GenerateTokenPair(sub, nil)
	if err != nil {
		return err
	}

	deliverer := h.deliverer
	if h.resolver != nil {
		if resolved := h.resolver.Resolve(c); resolved != nil {
			deliverer = resolved
		}
	}
	if deliverer == nil {
		deliverer = JSONPairDeliverer()
	}
	return deliverer.Deliver(c, pair)
}
