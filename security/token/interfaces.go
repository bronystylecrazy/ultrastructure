package token

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

// Manager is the auth token contract used by handlers/middleware wiring.
// Service is the current JWT implementation; future PASETO impl can satisfy this too.
type Manager interface {
	GenerateTokenPair(subject string, additionalAccessClaims map[string]any) (*TokenPair, error)
	RefreshTokenPair(refreshToken string, additionalAccessClaims map[string]any) (*TokenPair, error)
	ValidateToken(tokenValue string, expectedType string) (Claims, error)
	AccessMiddleware(extractors ...Extractor) fiber.Handler
	RefreshMiddleware(extractors ...Extractor) fiber.Handler
	RevokeToken(ctx context.Context, tokenValue string) error
	RevokeFromContext(c fiber.Ctx) error
}
