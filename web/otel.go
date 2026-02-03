package web

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	fiberzap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func NopOtelMiddleware() di.Node {
	return di.Replace(&OtelMiddleware{
		Telemetry: otel.Nop(),
	})
}

type OtelMiddleware struct {
	otel.Telemetry
	config *otel.Config
	lp     *otel.LoggerProvider
}

func NewOtelMiddleware(config otel.Config, lp *otel.LoggerProvider) *OtelMiddleware {
	return &OtelMiddleware{
		Telemetry: otel.Nop(),
		config:    &config,
		lp:        lp,
	}
}

func (h *OtelMiddleware) Handle(r fiber.Router) {
	r.Use(h.provideTracer)
	r.Use(fiberzap.New(fiberzap.Config{
		Logger: h.Obs.Logger,
		FieldsFunc: func(c fiber.Ctx) []zap.Field {
			return otel.ContextFunc(c.Context())
		},
	}))
}

func (h *OtelMiddleware) provideTracer(c fiber.Ctx) error {
	ctx, span := h.Obs.Start(c.Context(), fmt.Sprintf("%s %s", c.Method(), c.Path()))
	defer span.End()

	c.SetContext(ctx)

	return c.Next()
}
