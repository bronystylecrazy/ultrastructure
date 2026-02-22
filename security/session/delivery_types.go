package session

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
