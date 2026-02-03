package buildinfo

import (
	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/gofiber/fiber/v3"
)

const DefaultPath = "/api"

type Option func(*Handler)

type Handler struct {
	defaultPath string
}

type BuildInfoResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildDate   string `json:"buildDate"`
}

func New(opts ...Option) *Handler {
	return NewHandler(opts...)
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

func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		defaultPath: DefaultPath,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

func (h *Handler) Handle(r fiber.Router) {
	r.Get(h.defaultPath, func(c fiber.Ctx) error {
		resp := BuildInfoResponse{
			Name:        us.Name,
			Description: us.Description,
			Version:     us.Version,
			Commit:      us.Commit,
			BuildDate:   us.BuildDate,
		}
		return c.JSON(resp)
	})
}
