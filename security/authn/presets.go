package authn

import (
	apikey "github.com/bronystylecrazy/ultrastructure/security/apikey"
	token "github.com/bronystylecrazy/ultrastructure/security/token"
	"github.com/gofiber/fiber/v3"
)

func UserOnly(user token.Manager, modes ...ErrorMode) fiber.Handler {
	return AnyWithMode(resolveErrorMode(modes...), UserTokenAuthenticator(user))
}

func APIKeyOnly(app apikey.Manager, modes ...ErrorMode) fiber.Handler {
	return AnyWithMode(resolveErrorMode(modes...), APIKeyAuthenticator(app))
}

func UserAndAPIKey(user token.Manager, app apikey.Manager, modes ...ErrorMode) fiber.Handler {
	return AnyWithMode(
		resolveErrorMode(modes...),
		UserTokenAuthenticator(user),
		APIKeyAuthenticator(app),
	)
}

func resolveErrorMode(modes ...ErrorMode) ErrorMode {
	if len(modes) == 0 {
		return ErrorModeFailFast
	}
	switch modes[0] {
	case ErrorModeBestEffort:
		return ErrorModeBestEffort
	default:
		return ErrorModeFailFast
	}
}
