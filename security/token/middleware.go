package token

import (
	"errors"
	"fmt"

	httpx "github.com/bronystylecrazy/ultrastructure/security/internal/httpx"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	jwtgo "github.com/golang-jwt/jwt/v5"
)

var ErrTokenMissingInContext = errors.New("token: token missing from fiber context")

func (s *Service) AccessMiddleware(exs ...Extractor) fiber.Handler {
	if len(exs) == 0 {
		return s.tokenMiddleware(TokenTypeAccess, s.defaultAccess())
	}
	return s.tokenMiddleware(TokenTypeAccess, chainOrDefault(exs...))
}

func (s *Service) RefreshMiddleware(exs ...Extractor) fiber.Handler {
	if len(exs) == 0 {
		return s.tokenMiddleware(TokenTypeRefresh, s.defaultRefresh())
	}
	return s.tokenMiddleware(TokenTypeRefresh, chainOrDefault(exs...))
}

func (s *Service) tokenMiddleware(expectedType string, extractor Extractor) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: s.signingKey},
		Extractor:  extractor,
		SuccessHandler: func(c fiber.Ctx) error {
			claims, err := ClaimsFromContext(c)
			if err != nil {
				return unauthorized(c, err)
			}
			if claims.TokenType != expectedType {
				return unauthorized(c, fmt.Errorf("%w: got=%s want=%s", ErrInvalidTokenType, claims.TokenType, expectedType))
			}
			if err := s.ensureNotRevoked(c.Context(), claims); err != nil {
				return unauthorized(c, err)
			}
			return c.Next()
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return unauthorized(c, err)
		},
	})
}

func ClaimsFromContext(c fiber.Ctx) (Claims, error) {
	token := jwtware.FromContext(c)
	if token == nil {
		return Claims{}, ErrTokenMissingInContext
	}
	mapClaims, ok := token.Claims.(jwtgo.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidClaims
	}
	return claimsFromJWT(mapClaims), nil
}

func SubjectFromContext(c fiber.Ctx) (string, error) {
	claims, err := ClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrMissingTokenSub
	}
	return claims.Subject, nil
}

func unauthorized(c fiber.Ctx, err error) error {
	return httpx.Unauthorized(c, err.Error())
}

func chainOrDefault(exs ...Extractor) Extractor {
	if len(exs) == 0 {
		return FromAuthHeader("Bearer")
	}
	return Chain(exs...)
}
