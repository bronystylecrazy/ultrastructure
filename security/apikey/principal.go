package apikey

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

type Principal struct {
	Type     string            `json:"type"`
	AppID    string            `json:"app_id"`
	KeyID    string            `json:"key_id"`
	Scopes   []string          `json:"scopes"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

const PrincipalTypeAPIKey = "apikey"

type principalContextKey struct{}

const principalLocalsKey = "us.apikey.principal"

func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	v := ctx.Value(principalContextKey{})
	p, ok := v.(*Principal)
	return p, ok && p != nil
}

func SetPrincipalLocals(c fiber.Ctx, principal *Principal) {
	c.Locals(principalLocalsKey, principal)
}

func PrincipalFromLocals(c fiber.Ctx) (*Principal, bool) {
	v := c.Locals(principalLocalsKey)
	p, ok := v.(*Principal)
	return p, ok && p != nil
}
