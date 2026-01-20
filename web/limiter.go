package web

import (
	"time"

	"github.com/bronystylecrazy/ultrastructure/build"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

type LimitterHandler interface {
	Handler
}

var NopLimitterHandler LimitterHandler = &NopHandler{}

type limitterHandler struct{}

func NewLimitterHandler() LimitterHandler {
	return &limitterHandler{}
}

func (h *limitterHandler) Handle(app App) {
	app.Use("/api", limiter.New(limiter.Config{
		Max:               60,
		Expiration:        30 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
		Next: func(c *fiber.Ctx) bool {
			return build.IsDevelopment()
		},
	}))
}
