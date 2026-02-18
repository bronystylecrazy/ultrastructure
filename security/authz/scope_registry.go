package authz

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/web"
)

type ScopeDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

type ScopeRegistry struct {
	defs map[string]ScopeDefinition
}

func NewScopeRegistry(defs ...ScopeDefinition) (*ScopeRegistry, error) {
	out := &ScopeRegistry{
		defs: make(map[string]ScopeDefinition, len(defs)),
	}
	for _, def := range defs {
		name := strings.TrimSpace(def.Name)
		if name == "" {
			return nil, fmt.Errorf("authz: scope definition name is required")
		}
		def.Name = name
		def.Description = strings.TrimSpace(def.Description)
		def.Owner = strings.TrimSpace(def.Owner)
		def.Replacement = strings.TrimSpace(def.Replacement)

		if prev, ok := out.defs[name]; ok && prev != def {
			return nil, fmt.Errorf("authz: duplicate scope definition with conflicting metadata: %s", name)
		}
		out.defs[name] = def
	}
	return out, nil
}

func (r *ScopeRegistry) Has(scope string) bool {
	if r == nil {
		return false
	}
	_, ok := r.defs[strings.TrimSpace(scope)]
	return ok
}

func (r *ScopeRegistry) All() []ScopeDefinition {
	if r == nil || len(r.defs) == 0 {
		return nil
	}
	out := make([]ScopeDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

type UnknownRouteScope struct {
	Method string
	Path   string
	Scheme string
	Scope  string
}

type UnknownRouteScopesError struct {
	Items []UnknownRouteScope
}

func (e *UnknownRouteScopesError) Error() string {
	if e == nil || len(e.Items) == 0 {
		return "authz: unknown route scopes"
	}
	parts := make([]string, 0, len(e.Items))
	for _, item := range e.Items {
		parts = append(parts, fmt.Sprintf("%s %s [%s] => %s", item.Method, item.Path, item.Scheme, item.Scope))
	}
	return "authz: unknown route scopes: " + strings.Join(parts, "; ")
}

func ValidateRouteScopes(registry *web.MetadataRegistry, scopeRegistry *ScopeRegistry) error {
	if registry == nil || scopeRegistry == nil {
		return nil
	}
	routes := registry.AllRoutes()
	if len(routes) == 0 {
		return nil
	}

	unknown := make([]UnknownRouteScope, 0, 8)
	for key, meta := range routes {
		if meta == nil || len(meta.Security) == 0 {
			continue
		}
		method, path := splitRouteKey(key)
		for _, req := range meta.Security {
			scheme := strings.TrimSpace(req.Scheme)
			for _, scope := range req.Scopes {
				scope = strings.TrimSpace(scope)
				if scope == "" {
					continue
				}
				if scopeRegistry.Has(scope) {
					continue
				}
				unknown = append(unknown, UnknownRouteScope{
					Method: method,
					Path:   path,
					Scheme: scheme,
					Scope:  scope,
				})
			}
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Slice(unknown, func(i, j int) bool {
		if unknown[i].Path == unknown[j].Path {
			if unknown[i].Method == unknown[j].Method {
				if unknown[i].Scheme == unknown[j].Scheme {
					return unknown[i].Scope < unknown[j].Scope
				}
				return unknown[i].Scheme < unknown[j].Scheme
			}
			return unknown[i].Method < unknown[j].Method
		}
		return unknown[i].Path < unknown[j].Path
	})
	return &UnknownRouteScopesError{Items: unknown}
}
