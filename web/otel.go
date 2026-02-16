package web

import (
	"fmt"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	fiberzap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
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
	r.Use(h.WrapObserver)
	r.Use(fiberzap.New(fiberzap.Config{
		Logger: h.Obs.Logger,
		FieldsFunc: func(c fiber.Ctx) []zap.Field {
			return otel.ContextFunc(c.Context())
		},
	}))
}

func (h *OtelMiddleware) WrapObserver(c fiber.Ctx) error {
	startedAt := time.Now()
	ctx, span := h.Obs.Start(c.Context(), fmt.Sprintf("%s %s", c.Method(), c.Path()))
	defer span.End()

	c.SetContext(ctx)

	err := c.Next()

	status := c.Response().StatusCode()
	if status <= 0 {
		if err != nil {
			status = fiber.StatusInternalServerError
		} else {
			status = fiber.StatusOK
		}
	}

	route := c.FullPath()
	if route == "" {
		route = c.Path()
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", c.Method()),
		attribute.String("http.route", route),
		attribute.Int("http.response.status_code", status),
	}
	span.AddCounter(ctx, "requests.total", 1, attrs...)
	span.RecordHistogram(ctx, "request.duration", float64(time.Since(startedAt))/float64(time.Millisecond), attrs...)

	return err
}
