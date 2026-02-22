package session

import "github.com/gofiber/fiber/v3"

func JSONPairDeliverer() PairDeliverer {
	return PairDeliverFunc(func(c fiber.Ctx, pair *TokenPair) error {
		return c.JSON(pair)
	})
}
