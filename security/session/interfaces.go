package session

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Issuer interface {
	Generate(subject string, opts ...GenerateOption) (*TokenPair, error)
}

type Rotator interface {
	RotateRefresh(refreshToken string, opts ...GenerateOption) (*TokenPair, error)
	RotateAccess(accessToken string, opts ...GenerateOption) (string, time.Time, error)
}

type Validator interface {
	Validate(tokenValue string, expectedType string) (Claims, error)
}

type Revoker interface {
	Revoke(ctx context.Context, tokenValue string) error
	RevokeFromContext(c fiber.Ctx) error
}

// ContextTokenRevoker supports explicit token-type revocation from request context.
type ContextTokenRevoker interface {
	RevokeAccessFromContext(c fiber.Ctx) error
	RevokeRefreshFromContext(c fiber.Ctx) error
}

type MiddlewareFactory interface {
	AccessMiddleware(extractors ...Extractor) fiber.Handler
	RefreshMiddleware(extractors ...Extractor) fiber.Handler
}

// Manager is the default session contract composed from focused interfaces.
type Manager interface {
	Issuer
	Rotator
	Validator
	Revoker
	MiddlewareFactory
}
