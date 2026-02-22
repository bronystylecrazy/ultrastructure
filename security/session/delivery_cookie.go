package session

import "github.com/gofiber/fiber/v3"

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
