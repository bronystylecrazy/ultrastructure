package authz

import (
	"strings"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

const defaultScopeCatalogPath = "/api/v1/authz/scopes"

type ScopeCatalogHandler struct {
	path           string
	registry       *web.MetadataRegistry
	scopeRegistry  *ScopeRegistry
	policyRegistry *PolicyRegistry
}

func NewScopeCatalogHandler(registry *web.MetadataRegistry) *ScopeCatalogHandler {
	return &ScopeCatalogHandler{
		path:     defaultScopeCatalogPath,
		registry: registry,
	}
}

func (h *ScopeCatalogHandler) WithPath(path string) *ScopeCatalogHandler {
	path = strings.TrimSpace(path)
	if path != "" {
		h.path = path
	}
	return h
}

func (h *ScopeCatalogHandler) WithScopeRegistry(scopeRegistry *ScopeRegistry) *ScopeCatalogHandler {
	h.scopeRegistry = scopeRegistry
	return h
}

func (h *ScopeCatalogHandler) WithPolicyRegistry(policyRegistry *PolicyRegistry) *ScopeCatalogHandler {
	h.policyRegistry = policyRegistry
	return h
}

func (h *ScopeCatalogHandler) Handle(r web.Router) {
	r.Get(h.path, h.List).Apply(
		web.Tag("Authz"),
		web.Name("Authz_GetScopeCatalog"),
		web.Summary("Get authorization catalog"),
		web.Ok[ScopeCatalog](),
	)
}

func (h *ScopeCatalogHandler) List(c fiber.Ctx) error {
	return c.JSON(BuildScopeCatalogWithGovernance(h.registry, h.scopeRegistry, h.policyRegistry))
}
