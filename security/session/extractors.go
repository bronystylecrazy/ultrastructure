package session

import (
	"github.com/gofiber/fiber/v3"
	fiberextractors "github.com/gofiber/fiber/v3/extractors"
)

type Extractor = fiberextractors.Extractor

func FromAuthHeader(authScheme string) Extractor {
	return fiberextractors.FromAuthHeader(authScheme)
}

func FromCookie(key string) Extractor {
	return fiberextractors.FromCookie(key)
}

func FromParam(param string) Extractor {
	return fiberextractors.FromParam(param)
}

func FromForm(param string) Extractor {
	return fiberextractors.FromForm(param)
}

func FromHeader(header string) Extractor {
	return fiberextractors.FromHeader(header)
}

func FromQuery(param string) Extractor {
	return fiberextractors.FromQuery(param)
}

func FromCustom(key string, fn func(fiber.Ctx) (string, error)) Extractor {
	return fiberextractors.FromCustom(key, fn)
}

func Chain(extractors ...Extractor) Extractor {
	return fiberextractors.Chain(extractors...)
}
