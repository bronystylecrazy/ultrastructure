package authz

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
)

type PolicyDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	Replacement string `json:"replacement,omitempty"`

	PrincipalType authn.PrincipalType `json:"principal_type,omitempty"`
	AnyScopes     []string            `json:"any_scopes,omitempty"`
	AllScopes     []string            `json:"all_scopes,omitempty"`
	Roles         []string            `json:"roles,omitempty"`
}

type PolicyRegistry struct {
	defs map[string]PolicyDefinition
}

func NewPolicyRegistry(defs ...PolicyDefinition) (*PolicyRegistry, error) {
	out := &PolicyRegistry{
		defs: make(map[string]PolicyDefinition, len(defs)),
	}
	for _, def := range defs {
		normalized, err := normalizePolicyDefinition(def)
		if err != nil {
			return nil, err
		}
		if prev, ok := out.defs[normalized.Name]; ok && !equalPolicyDefinition(prev, normalized) {
			return nil, fmt.Errorf("authz: duplicate policy definition with conflicting metadata: %s", normalized.Name)
		}
		out.defs[normalized.Name] = normalized
	}
	return out, nil
}

func (r *PolicyRegistry) Has(name string) bool {
	if r == nil {
		return false
	}
	_, ok := r.defs[strings.TrimSpace(name)]
	return ok
}

func (r *PolicyRegistry) Get(name string) (PolicyDefinition, bool) {
	if r == nil {
		return PolicyDefinition{}, false
	}
	def, ok := r.defs[strings.TrimSpace(name)]
	return def, ok
}

func (r *PolicyRegistry) All() []PolicyDefinition {
	if r == nil || len(r.defs) == 0 {
		return nil
	}
	out := make([]PolicyDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func normalizePolicyDefinition(def PolicyDefinition) (PolicyDefinition, error) {
	def.Name = strings.TrimSpace(def.Name)
	if def.Name == "" {
		return PolicyDefinition{}, fmt.Errorf("authz: policy definition name is required")
	}
	def.Description = strings.TrimSpace(def.Description)
	def.Owner = strings.TrimSpace(def.Owner)
	def.Replacement = strings.TrimSpace(def.Replacement)
	def.AnyScopes = normalizeStringList(def.AnyScopes)
	def.AllScopes = normalizeStringList(def.AllScopes)
	def.Roles = normalizeStringList(def.Roles)
	return def, nil
}

func normalizeStringList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func equalPolicyDefinition(a, b PolicyDefinition) bool {
	return a.Name == b.Name &&
		a.Description == b.Description &&
		a.Owner == b.Owner &&
		a.Deprecated == b.Deprecated &&
		a.Replacement == b.Replacement &&
		a.PrincipalType == b.PrincipalType &&
		reflect.DeepEqual(a.AnyScopes, b.AnyScopes) &&
		reflect.DeepEqual(a.AllScopes, b.AllScopes) &&
		reflect.DeepEqual(a.Roles, b.Roles)
}
