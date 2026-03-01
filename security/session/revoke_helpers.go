package session

import "github.com/gofiber/fiber/v3"

func extractRefreshTokenForRevoke(c fiber.Ctx) (string, error) {
	return Chain(
		FromHeader("X-Refresh-Token"),
		FromCookie("refresh_token"),
		FromQuery("refresh_token"),
		FromForm("refresh_token"),
		FromParam("refresh_token"),
	).Extract(c)
}
