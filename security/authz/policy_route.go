package authz

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	"github.com/bronystylecrazy/ultrastructure/web"
)

const (
	defaultUserPolicyScheme = "BearerAuth"
	defaultAppPolicyScheme  = "ApiKeyAuth"
)

var (
	policyExpansionRegistryMu sync.RWMutex
	policyExpansionRegistry   *PolicyRegistry
)

func Policy(name string) web.RouteOption {
	return Policies(name)
}

func Policies(names ...string) web.RouteOption {
	return func(b *web.RouteBuilder) *web.RouteBuilder {
		b = b.Policies(names...)
		reg := getPolicyExpansionRegistry()
		if reg == nil {
			return b
		}
		for _, name := range normalizeStringList(names) {
			def, ok := reg.Get(name)
			if !ok {
				continue
			}
			for _, req := range securityRequirementsForPolicy(def) {
				b = b.Scopes(req.Scheme, req.Scopes...)
			}
		}
		return b
	}
}

func setPolicyExpansionRegistry(reg *PolicyRegistry) {
	policyExpansionRegistryMu.Lock()
	defer policyExpansionRegistryMu.Unlock()
	policyExpansionRegistry = reg
}

func getPolicyExpansionRegistry() *PolicyRegistry {
	policyExpansionRegistryMu.RLock()
	defer policyExpansionRegistryMu.RUnlock()
	return policyExpansionRegistry
}

type UnknownRoutePolicy struct {
	Method string
	Path   string
	Policy string
}

type UnknownRoutePoliciesError struct {
	Items []UnknownRoutePolicy
}

func (e *UnknownRoutePoliciesError) Error() string {
	if e == nil || len(e.Items) == 0 {
		return "authz: unknown route policies"
	}
	parts := make([]string, 0, len(e.Items))
	for _, item := range e.Items {
		parts = append(parts, fmt.Sprintf("%s %s => %s", item.Method, item.Path, item.Policy))
	}
	return "authz: unknown route policies: " + strings.Join(parts, "; ")
}

func ValidateRoutePolicies(registry *web.MetadataRegistry, policyRegistry *PolicyRegistry) error {
	if registry == nil || policyRegistry == nil {
		return nil
	}
	routes := registry.AllRoutes()
	if len(routes) == 0 {
		return nil
	}

	unknown := make([]UnknownRoutePolicy, 0, 8)
	for key, meta := range routes {
		if meta == nil || len(meta.Policies) == 0 {
			continue
		}
		method, path := splitRouteKey(key)
		for _, name := range normalizeStringList(meta.Policies) {
			if policyRegistry.Has(name) {
				continue
			}
			unknown = append(unknown, UnknownRoutePolicy{
				Method: method,
				Path:   path,
				Policy: name,
			})
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Slice(unknown, func(i, j int) bool {
		if unknown[i].Path == unknown[j].Path {
			if unknown[i].Method == unknown[j].Method {
				return unknown[i].Policy < unknown[j].Policy
			}
			return unknown[i].Method < unknown[j].Method
		}
		return unknown[i].Path < unknown[j].Path
	})
	return &UnknownRoutePoliciesError{Items: unknown}
}

func ExpandRoutePolicies(registry *web.MetadataRegistry, policyRegistry *PolicyRegistry) {
	if registry == nil || policyRegistry == nil {
		return
	}
	routes := registry.AllRoutes()
	for _, meta := range routes {
		if meta == nil || len(meta.Policies) == 0 {
			continue
		}
		for _, name := range normalizeStringList(meta.Policies) {
			def, ok := policyRegistry.Get(name)
			if !ok {
				continue
			}
			for _, req := range securityRequirementsForPolicy(def) {
				appendSecurityRequirement(meta, req)
			}
		}
	}
}

func securityRequirementsForPolicy(def PolicyDefinition) []web.SecurityRequirement {
	schemes := []string{defaultUserPolicyScheme, defaultAppPolicyScheme}
	switch def.PrincipalType {
	case authn.PrincipalUser:
		schemes = []string{defaultUserPolicyScheme}
	case authn.PrincipalApp:
		schemes = []string{defaultAppPolicyScheme}
	}

	allScopes := normalizeStringList(def.AllScopes)
	anyScopes := normalizeStringList(def.AnyScopes)
	out := make([]web.SecurityRequirement, 0, len(schemes)*(len(allScopes)+len(anyScopes)))
	for _, scheme := range schemes {
		if len(allScopes) > 0 {
			out = append(out, web.SecurityRequirement{
				Scheme: scheme,
				Scopes: append([]string(nil), allScopes...),
			})
		}
		for _, scope := range anyScopes {
			out = append(out, web.SecurityRequirement{
				Scheme: scheme,
				Scopes: []string{scope},
			})
		}
	}
	return out
}

func appendSecurityRequirement(meta *web.RouteMetadata, req web.SecurityRequirement) {
	if meta == nil {
		return
	}
	scheme := strings.TrimSpace(req.Scheme)
	if scheme == "" {
		return
	}
	scopes := normalizeStringList(req.Scopes)
	if len(scopes) == 0 {
		return
	}
	req.Scheme = scheme
	req.Scopes = scopes
	key := requirementKey(req)
	for _, existing := range meta.Security {
		if requirementKey(existing) == key {
			return
		}
	}
	meta.Security = append(meta.Security, req)
}

func requirementKey(req web.SecurityRequirement) string {
	scheme := strings.TrimSpace(req.Scheme)
	scopes := normalizeStringList(req.Scopes)
	return scheme + "|" + strings.Join(scopes, ",")
}
