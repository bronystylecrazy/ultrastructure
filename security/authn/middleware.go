package authn

import (
	"strings"

	apikey "github.com/bronystylecrazy/ultrastructure/security/apikey"
	httpx "github.com/bronystylecrazy/ultrastructure/security/internal/httpx"
	"github.com/bronystylecrazy/ultrastructure/security/session"
	"github.com/gofiber/fiber/v3"
)

type Authenticator interface {
	// Authenticate returns:
	// - principal when auth succeeds
	// - matched=true when credential source was present and processed
	// - err when processing/validation failed
	Authenticate(c fiber.Ctx) (*Principal, bool, error)
}

type AuthenticatorFunc func(c fiber.Ctx) (*Principal, bool, error)

func (f AuthenticatorFunc) Authenticate(c fiber.Ctx) (*Principal, bool, error) {
	return f(c)
}

func Any(authenticators ...Authenticator) fiber.Handler {
	return AnyWithMode(ErrorModeFailFast, authenticators...)
}

func AnyWithMode(mode ErrorMode, authenticators ...Authenticator) fiber.Handler {
	return func(c fiber.Ctx) error {
		principals := make([]*Principal, 0, len(authenticators))
		for _, a := range authenticators {
			if a == nil {
				continue
			}
			p, matched, err := a.Authenticate(c)
			if !matched {
				continue
			}
			if err != nil {
				if mode == ErrorModeBestEffort {
					continue
				}
				return httpx.Unauthorized(c, "unauthorized")
			}
			if p == nil {
				continue
			}
			principals = append(principals, p)
		}

		if len(principals) == 0 {
			return httpx.Unauthorized(c, "unauthorized")
		}

		primary := principals[0]
		ctx := WithPrincipals(c.Context(), principals)
		ctx = WithPrincipal(ctx, primary)
		c.SetContext(ctx)
		SetPrincipalsLocals(c, principals)
		SetPrincipalLocals(c, primary)
		return c.Next()
	}
}

func UserTokenAuthenticator(user session.Validator) Authenticator {
	return AuthenticatorFunc(func(c fiber.Ctx) (*Principal, bool, error) {
		if user == nil {
			return nil, false, nil
		}
		auth := strings.TrimSpace(c.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return nil, false, nil
		}
		raw := strings.TrimSpace(auth[len("Bearer "):])
		claims, err := user.Validate(raw, session.TokenTypeAccess)
		if err != nil {
			return nil, true, err
		}
		return &Principal{
			Type:    PrincipalUser,
			Subject: claimString(claims.Values, "sub"),
			Roles:   claimRoles(claims.Values),
			Scopes:  claimScopes(claims.Values),
		}, true, nil
	})
}

func APIKeyAuthenticator(app apikey.Manager) Authenticator {
	return AuthenticatorFunc(func(c fiber.Ctx) (*Principal, bool, error) {
		if app == nil {
			return nil, false, nil
		}
		auth := strings.TrimSpace(c.Get("Authorization"))
		rawKey := extractAPIKey(auth, c.Get("X-API-Key"))
		if rawKey == "" {
			return nil, false, nil
		}
		ap, err := app.ValidateRawKey(c.Context(), rawKey)
		if err != nil {
			return nil, true, err
		}
		return &Principal{
			Type:   PrincipalApp,
			AppID:  ap.AppID,
			KeyID:  ap.KeyID,
			Scopes: append([]string(nil), ap.Scopes...),
		}, true, nil
	})
}
