package session

import "github.com/gofiber/fiber/v3"

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
