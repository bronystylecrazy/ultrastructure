package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/gofiber/fiber/v3"
)

const DefaultBuildInfoPath = "/api"

type BuildInfoOption func(*BuildInfoHandler)

type BuildInfoHandler struct {
	defaultPath string
}

type BuildInfoResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildDate   string `json:"buildDate"`
}

func UseBuildInfo(opts ...BuildInfoOption) di.Node {
	return di.Options(
		di.Provide(func() *BuildInfoHandler {
			return NewBuildInfoHandler(opts...)
		}),
	)
}

func WithDefaultPath(path ...string) BuildInfoOption {
	return func(h *BuildInfoHandler) {
		if len(path) == 0 {
			h.defaultPath = DefaultBuildInfoPath
			return
		}
		h.defaultPath = path[0]
	}
}

func NewBuildInfoHandler(opts ...BuildInfoOption) *BuildInfoHandler {
	h := &BuildInfoHandler{
		defaultPath: DefaultBuildInfoPath,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

func (h *BuildInfoHandler) Handle(r Router) {
	r.Get(h.defaultPath, func(c fiber.Ctx) error {
		return c.JSON(BuildInfoResponse{
			Name:        meta.Name,
			Description: meta.Description,
			Version:     meta.Version,
			Commit:      meta.Commit,
			BuildDate:   meta.BuildDate,
		})
	})
}
