package authn

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

type PrincipalType string

const (
	PrincipalUser PrincipalType = "user"
	PrincipalApp  PrincipalType = "app"
)

type Principal struct {
	Type    PrincipalType `json:"type"`
	Subject string        `json:"subject,omitempty"`
	AppID   string        `json:"app_id,omitempty"`
	KeyID   string        `json:"key_id,omitempty"`
	Scopes  []string      `json:"scopes,omitempty"`
	Roles   []string      `json:"roles,omitempty"`
}

type principalContextKey struct{}
type principalsContextKey struct{}

const principalLocalsKey = "us.authn.principal"
const principalsLocalsKey = "us.authn.principals"

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	v := ctx.Value(principalContextKey{})
	p, ok := v.(*Principal)
	return p, ok && p != nil
}

func WithPrincipals(ctx context.Context, principals []*Principal) context.Context {
	return context.WithValue(ctx, principalsContextKey{}, principals)
}

func PrincipalsFromContext(ctx context.Context) ([]*Principal, bool) {
	v := ctx.Value(principalsContextKey{})
	p, ok := v.([]*Principal)
	return p, ok && len(p) > 0
}

func SetPrincipalLocals(c fiber.Ctx, p *Principal) {
	c.Locals(principalLocalsKey, p)
}

func PrincipalFromLocals(c fiber.Ctx) (*Principal, bool) {
	v := c.Locals(principalLocalsKey)
	p, ok := v.(*Principal)
	return p, ok && p != nil
}

func SetPrincipalsLocals(c fiber.Ctx, principals []*Principal) {
	c.Locals(principalsLocalsKey, principals)
}

func PrincipalsFromLocals(c fiber.Ctx) ([]*Principal, bool) {
	v := c.Locals(principalsLocalsKey)
	p, ok := v.([]*Principal)
	return p, ok && len(p) > 0
}
