package token

import "github.com/gofiber/fiber/v3"

type PairDeliverer interface {
	Deliver(c fiber.Ctx, pair *TokenPair) error
}

type PairDelivererResolver interface {
	Resolve(c fiber.Ctx) PairDeliverer
}

type PairDelivererResolverFunc func(c fiber.Ctx) PairDeliverer

func (f PairDelivererResolverFunc) Resolve(c fiber.Ctx) PairDeliverer {
	return f(c)
}

type PairDeliverFunc func(c fiber.Ctx, pair *TokenPair) error

func (f PairDeliverFunc) Deliver(c fiber.Ctx, pair *TokenPair) error {
	return f(c, pair)
}

func JSONPairDeliverer() PairDeliverer {
	return PairDeliverFunc(func(c fiber.Ctx, pair *TokenPair) error {
		return c.JSON(pair)
	})
}

func WebCookieOrJSONPairDelivererResolver() PairDelivererResolver {
	return PairDelivererResolverFunc(func(c fiber.Ctx) PairDeliverer {
		if c.Get("X-Client-Type") == "web" {
			return CookiePairDeliverer(CookiePairDelivererConfig{
				AccessCookieTemplate:  fiber.Cookie{HTTPOnly: true, Secure: true, Path: "/"},
				RefreshCookieTemplate: fiber.Cookie{HTTPOnly: true, Secure: true, Path: "/"},
			})
		}
		return JSONPairDeliverer()
	})
}

func JSONOnlyPairDelivererResolver() PairDelivererResolver {
	return PairDelivererResolverFunc(func(c fiber.Ctx) PairDeliverer {
		return JSONPairDeliverer()
	})
}

func DefaultRefreshPairDelivererResolver() PairDelivererResolver {
	return WebCookieOrJSONPairDelivererResolver()
}

type CookiePairDelivererConfig struct {
	AccessTokenCookieName  string
	RefreshTokenCookieName string
	AccessCookieTemplate   fiber.Cookie
	RefreshCookieTemplate  fiber.Cookie
	IncludeJSONBody        bool
	StatusCode             int
}

func CookiePairDeliverer(cfg CookiePairDelivererConfig) PairDeliverer {
	if cfg.AccessTokenCookieName == "" {
		cfg.AccessTokenCookieName = "access_token"
	}
	if cfg.RefreshTokenCookieName == "" {
		cfg.RefreshTokenCookieName = "refresh_token"
	}
	if cfg.StatusCode == 0 {
		if cfg.IncludeJSONBody {
			cfg.StatusCode = fiber.StatusOK
		} else {
			cfg.StatusCode = fiber.StatusNoContent
		}
	}

	return PairDeliverFunc(func(c fiber.Ctx, pair *TokenPair) error {
		accessCookie := cfg.AccessCookieTemplate
		accessCookie.Name = cfg.AccessTokenCookieName
		accessCookie.Value = pair.AccessToken
		if accessCookie.Expires.IsZero() {
			accessCookie.Expires = pair.AccessExpiresAt
		}

		refreshCookie := cfg.RefreshCookieTemplate
		refreshCookie.Name = cfg.RefreshTokenCookieName
		refreshCookie.Value = pair.RefreshToken
		if refreshCookie.Expires.IsZero() {
			refreshCookie.Expires = pair.RefreshExpiresAt
		}

		c.Cookie(&accessCookie)
		c.Cookie(&refreshCookie)

		if cfg.IncludeJSONBody {
			return c.Status(cfg.StatusCode).JSON(pair)
		}
		return c.SendStatus(cfg.StatusCode)
	})
}

type HeaderPairDelivererConfig struct {
	AccessTokenHeader  string
	RefreshTokenHeader string
	IncludeJSONBody    bool
	StatusCode         int
}

func HeaderPairDeliverer(cfg HeaderPairDelivererConfig) PairDeliverer {
	if cfg.AccessTokenHeader == "" {
		cfg.AccessTokenHeader = "X-Access-Token"
	}
	if cfg.RefreshTokenHeader == "" {
		cfg.RefreshTokenHeader = "X-Refresh-Token"
	}
	if cfg.StatusCode == 0 {
		if cfg.IncludeJSONBody {
			cfg.StatusCode = fiber.StatusOK
		} else {
			cfg.StatusCode = fiber.StatusNoContent
		}
	}

	return PairDeliverFunc(func(c fiber.Ctx, pair *TokenPair) error {
		c.Set(cfg.AccessTokenHeader, pair.AccessToken)
		c.Set(cfg.RefreshTokenHeader, pair.RefreshToken)
		if cfg.IncludeJSONBody {
			return c.Status(cfg.StatusCode).JSON(pair)
		}
		return c.SendStatus(cfg.StatusCode)
	})
}
