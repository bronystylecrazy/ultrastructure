package authz

import (
	"context"
	"sort"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"go.uber.org/fx"
)

func UseScopeCatalogRoute(path string) di.Node {
	return di.Provide(func(container *web.RegistryContainer, scopeRegistry *ScopeRegistry, policyRegistry *PolicyRegistry) *ScopeCatalogHandler {
		registry := web.GetGlobalRegistry()
		if container != nil && container.Metadata != nil {
			registry = container.Metadata
		}
		return NewScopeCatalogHandler(registry).
			WithPath(path).
			WithScopeRegistry(scopeRegistry).
			WithPolicyRegistry(policyRegistry)
	}, di.Params(di.Optional(), di.Optional(), di.Optional()))
}

func UseScopeGovernance(defs ...ScopeDefinition) di.Node {
	registerScopeEnum(defs...)
	return di.Options(
		di.Provide(func() (*ScopeRegistry, error) {
			return NewScopeRegistry(defs...)
		}),
		di.Invoke(func(lc fx.Lifecycle, container *web.RegistryContainer, scopeRegistry *ScopeRegistry) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					registry := web.GetGlobalRegistry()
					if container != nil && container.Metadata != nil {
						registry = container.Metadata
					}
					return ValidateRouteScopes(registry, scopeRegistry)
				},
			})
		}, di.Params(``, di.Optional(), ``)),
	)
}

func UsePolicyGovernance(defs ...PolicyDefinition) di.Node {
	registerPolicyEnum(defs...)
	reg, err := NewPolicyRegistry(defs...)
	if err == nil {
		setPolicyExpansionRegistry(reg)
	}
	return di.Options(
		di.Provide(func() (*PolicyRegistry, error) {
			return NewPolicyRegistry(defs...)
		}),
		di.Invoke(func(lc fx.Lifecycle, container *web.RegistryContainer, policyRegistry *PolicyRegistry, scopeRegistry *ScopeRegistry) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					registry := web.GetGlobalRegistry()
					if container != nil && container.Metadata != nil {
						registry = container.Metadata
					}
					if err := ValidateRoutePolicies(registry, policyRegistry); err != nil {
						return err
					}
					ExpandRoutePolicies(registry, policyRegistry)
					if scopeRegistry != nil {
						if err := ValidateRouteScopes(registry, scopeRegistry); err != nil {
							return err
						}
					}
					return nil
				},
			})
		}, di.Params(``, di.Optional(), ``, di.Optional())),
	)
}

func registerScopeEnum(defs ...ScopeDefinition) {
	if len(defs) == 0 {
		return
	}
	seen := map[ScopeName]struct{}{}
	out := make([]ScopeName, 0, len(defs))
	for _, def := range defs {
		name := ScopeName(def.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	web.RegisterEnum[ScopeName](out...)
}

func registerPolicyEnum(defs ...PolicyDefinition) {
	if len(defs) == 0 {
		return
	}
	seen := map[PolicyName]struct{}{}
	out := make([]PolicyName, 0, len(defs))
	for _, def := range defs {
		name := PolicyName(def.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	web.RegisterEnum[PolicyName](out...)
}
