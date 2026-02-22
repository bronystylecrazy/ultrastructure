package session

import "github.com/gofiber/fiber/v3"

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
