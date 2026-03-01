package session

import "github.com/gofiber/fiber/v3"

func extractAccessTokenForRevoke(c fiber.Ctx) (string, error) {
	return Chain(
		FromAuthHeader("Bearer"),
		FromHeader("X-Access-Token"),
		FromCookie("access_token"),
		FromQuery("access_token"),
		FromForm("access_token"),
		FromParam("access_token"),
		FromCookie("token"),
		FromQuery("token"),
		FromForm("token"),
		FromParam("token"),
	).Extract(c)
}

func extractRefreshTokenForRevoke(c fiber.Ctx) (string, error) {
	return Chain(
		FromHeader("X-Refresh-Token"),
		FromCookie("refresh_token"),
		FromQuery("refresh_token"),
		FromForm("refresh_token"),
		FromParam("refresh_token"),
		FromCookie("token"),
		FromQuery("token"),
		FromForm("token"),
		FromParam("token"),
	).Extract(c)
}
