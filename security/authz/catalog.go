package authz

import (
	"sort"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/samber/lo"
)

type ScopeName string
type PolicyName string

type ScopeCatalog struct {
	Scopes            []ScopeName          `json:"scopes"`
	ScopeDefinitions  []ScopeDefinition    `json:"scope_definitions,omitempty"`
	PolicyDefinitions []PolicyDefinition   `json:"policy_definitions,omitempty"`
	Endpoints         []ScopeEndpointEntry `json:"endpoints"`
}

type ScopeEndpointEntry struct {
	Method      string       `json:"method"`
	Path        string       `json:"path"`
	OperationID string       `json:"operation_id,omitempty"`
	Schemes     []string     `json:"schemes,omitempty"`
	Scopes      []ScopeName  `json:"scopes,omitempty"`
	Policies    []PolicyName `json:"policies,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
}

func BuildScopeCatalog(registry *web.MetadataRegistry) ScopeCatalog {
	return BuildScopeCatalogWithGovernance(registry, nil, nil)
}

func BuildScopeCatalogWithRegistry(registry *web.MetadataRegistry, scopeRegistry *ScopeRegistry) ScopeCatalog {
	return BuildScopeCatalogWithGovernance(registry, scopeRegistry, nil)
}

func BuildScopeCatalogWithGovernance(registry *web.MetadataRegistry, scopeRegistry *ScopeRegistry, policyRegistry *PolicyRegistry) ScopeCatalog {
	if registry == nil {
		return ScopeCatalog{
			ScopeDefinitions:  scopeRegistry.All(),
			PolicyDefinitions: policyRegistry.All(),
		}
	}
	routes := registry.AllRoutes()
	scopeDefinitions := scopeRegistry.All()
	policyDefinitions := policyRegistry.All()

	scopeSet := lo.SliceToMap(
		lo.FilterMap(scopeDefinitions, func(def ScopeDefinition, _ int) (string, bool) {
			v := strings.TrimSpace(def.Name)
			return v, v != ""
		}),
		func(name string) (string, struct{}) { return name, struct{}{} },
	)
	endpoints := make([]ScopeEndpointEntry, 0, len(routes))

	for key, meta := range routes {
		method, path := splitRouteKey(key)
		if method == "" || path == "" || meta == nil {
			continue
		}

		schemes := make([]string, 0, len(meta.Security))
		scopes := make([]string, 0, len(meta.Security))
		policies := toPolicyNames(normalizeStringList(meta.Policies))
		for _, sec := range meta.Security {
			schemes = append(schemes, sec.Scheme)
			for _, scope := range sec.Scopes {
				scope = strings.TrimSpace(scope)
				if scope == "" {
					continue
				}
				scopes = append(scopes, scope)
				scopeSet[scope] = struct{}{}
			}
		}

		schemes = normalizeStringList(schemes)
		scopes = normalizeStringList(scopes)
		tags := append([]string(nil), meta.Tags...)
		sort.Strings(tags)

		endpoints = append(endpoints, ScopeEndpointEntry{
			Method:      method,
			Path:        path,
			OperationID: strings.TrimSpace(meta.OperationID),
			Schemes:     schemes,
			Scopes:      toScopeNames(scopes),
			Policies:    policies,
			Tags:        tags,
		})
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	scopes := lo.Keys(scopeSet)
	sort.Strings(scopes)

	return ScopeCatalog{
		Scopes:            toScopeNames(scopes),
		ScopeDefinitions:  scopeDefinitions,
		PolicyDefinitions: policyDefinitions,
		Endpoints:         endpoints,
	}
}

func toScopeNames(in []string) []ScopeName {
	if len(in) == 0 {
		return nil
	}
	out := lo.FilterMap(in, func(v string, _ int) (ScopeName, bool) {
		v = strings.TrimSpace(v)
		return ScopeName(v), v != ""
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func toPolicyNames(in []string) []PolicyName {
	if len(in) == 0 {
		return nil
	}
	out := lo.FilterMap(in, func(v string, _ int) (PolicyName, bool) {
		v = strings.TrimSpace(v)
		return PolicyName(v), v != ""
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func splitRouteKey(key string) (method, path string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
