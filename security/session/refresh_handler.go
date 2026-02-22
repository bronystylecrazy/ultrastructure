package session

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type SubjectResolver func(c fiber.Ctx) (string, error)

type RefreshHandler struct {
	service         Manager
	subjectResolver SubjectResolver
	path            string
	deliverer       PairDeliverer
	resolver        PairDelivererResolver
}

func NewRefreshHandler(service Manager, subjectResolver SubjectResolver) *RefreshHandler {
	return &RefreshHandler{
		service:         service,
		subjectResolver: subjectResolver,
		path:            "/api/v1/auth/refresh",
		deliverer:       JSONPairDeliverer(),
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

func (h *RefreshHandler) WithSubjectResolver(subjectResolver SubjectResolver) *RefreshHandler {
	if subjectResolver != nil {
		h.subjectResolver = subjectResolver
	}
	return h
}

func (h *RefreshHandler) Refresh(c fiber.Ctx) error {
	if h.subjectResolver == nil {
		return writeUnauthorized(c, ErrMissingRefreshSubjectResolver)
	}

	sub, err := h.subjectResolver(c)
	if err != nil {
		return writeUnauthorized(c, err)
	}

	pair, err := h.service.Generate(sub)
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
