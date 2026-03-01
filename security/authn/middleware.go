package authn

import (
	"context"
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
	return UserTokenAuthenticatorWithExtractors(user)
}

func UserTokenAuthenticatorWithExtractors(user session.Validator, extractors ...session.Extractor) Authenticator {
	return AuthenticatorFunc(func(c fiber.Ctx) (*Principal, bool, error) {
		if user == nil {
			return nil, false, nil
		}

		extractor := defaultUserTokenExtractor()
		if len(extractors) > 0 {
			extractor = session.Chain(extractors...)
		}

		raw, err := extractor.Extract(c)
		if err != nil || raw == "" {
			return nil, false, nil
		}

		claims, err := validateUserAccessToken(c.Context(), user, raw)
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

func validateUserAccessToken(ctx context.Context, user session.Validator, raw string) (session.Claims, error) {
	if active, ok := user.(session.ActiveValidator); ok {
		return active.ValidateActive(ctx, raw, session.TokenTypeAccess)
	}
	return user.Validate(raw, session.TokenTypeAccess)
}

func defaultUserTokenExtractor() session.Extractor {
	return session.Chain(
		session.FromAuthHeader("Bearer"),
		session.FromHeader("X-Access-Token"),
		session.FromCookie("access_token"),
		session.FromCookie("token"),
		session.FromQuery("access_token"),
		session.FromQuery("token"),
		session.FromForm("access_token"),
		session.FromForm("token"),
		session.FromParam("access_token"),
		session.FromParam("token"),
	)
}

func APIKeyAuthenticator(app apikey.Manager) Authenticator {
	return APIKeyAuthenticatorWithExtractors(app)
}

func APIKeyAuthenticatorWithExtractors(app apikey.Manager, extractors ...session.Extractor) Authenticator {
	return AuthenticatorFunc(func(c fiber.Ctx) (*Principal, bool, error) {
		if app == nil {
			return nil, false, nil
		}

		extractor := defaultAPIKeyExtractor()
		if len(extractors) > 0 {
			extractor = session.Chain(extractors...)
		}

		rawKey, err := extractor.Extract(c)
		if err != nil {
			return nil, false, nil
		}
		rawKey = strings.TrimSpace(rawKey)
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

func defaultAPIKeyExtractor() session.Extractor {
	return session.Chain(
		session.FromCustom("authorization_apikey_or_x_api_key", func(c fiber.Ctx) (string, error) {
			auth := strings.TrimSpace(c.Get("Authorization"))
			return extractAPIKey(auth, c.Get("X-API-Key")), nil
		}),
		session.FromHeader("X-API-Key"),
		session.FromCookie("api_key"),
		session.FromQuery("api_key"),
		session.FromForm("api_key"),
		session.FromParam("api_key"),
	)
}
