package web

import (
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	fiberzap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

func (h *OtelMiddleware) Handle(r Router) {
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
	ctx, span := h.Obs.Start(c.Context())
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

	// The route is only known after routing, so name the span once it has run.
	// A request no route matched is named by its method alone, following the
	// OpenTelemetry HTTP conventions, so unknown paths add no new span names.
	name := c.Method()
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", c.Method()),
		attribute.Int("http.response.status_code", status),
	}
	if c.Matched() && !c.IsMiddleware() {
		route := c.FullPath()
		name += " " + route
		attrs = append(attrs, attribute.String("http.route", route))
	}
	trace.SpanFromContext(ctx).SetName(name)
	span.AddCounter(ctx, "requests.total", 1, attrs...)
	span.RecordHistogram(ctx, "request.duration", float64(time.Since(startedAt))/float64(time.Millisecond), attrs...)

	return err
}
