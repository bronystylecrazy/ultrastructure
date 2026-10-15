package web

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
)

func newRecordingOtelMiddleware(t *testing.T) (*OtelMiddleware, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return &OtelMiddleware{
		Telemetry: otel.Telemetry{Obs: otel.NewObserver(zap.NewNop(), tp.Tracer("web"))},
	}, recorder
}

type otelRouteHandler struct {
	otel.Telemetry
}

func (h *otelRouteHandler) GetUser(c fiber.Ctx) error {
	_, span := h.Obs.Start(c.Context())
	defer span.End()
	return c.SendString(c.Params("id"))
}

func TestOtelMiddlewareNamesSpanAfterMatchedRoute(t *testing.T) {
	mw, recorder := newRecordingOtelMiddleware(t)
	handler := &otelRouteHandler{Telemetry: mw.Telemetry}
	app := fiber.New()
	app.Use(mw.WrapObserver)
	app.Get("/users/:id", handler.GetUser)

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/users/42", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	ended := recorder.Ended()
	if len(ended) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(ended))
	}
	handlerSpan, requestSpan := ended[0], ended[1]
	if got, want := requestSpan.Name(), "GET /users/:id"; got != want {
		t.Fatalf("request span name = %q, want %q", got, want)
	}
	if got, want := handlerSpan.Name(), "web.otelRouteHandler.GetUser"; got != want {
		t.Fatalf("handler span name = %q, want %q", got, want)
	}
	if handlerSpan.Parent().SpanID() != requestSpan.SpanContext().SpanID() {
		t.Fatal("handler span is not parented to the request span")
	}
}

func TestOtelMiddlewareNamesUnmatchedRequestAfterMethod(t *testing.T) {
	mw, recorder := newRecordingOtelMiddleware(t)
	app := fiber.New()
	app.Use(mw.WrapObserver)
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("root") })

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/missing", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 span, got %d", len(ended))
	}
	if got, want := ended[0].Name(), "GET"; got != want {
		t.Fatalf("request span name = %q, want %q", got, want)
	}
}

func TestOtelMiddlewareNamesRootRoute(t *testing.T) {
	mw, recorder := newRecordingOtelMiddleware(t)
	app := fiber.New()
	app.Use(mw.WrapObserver)
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("root") })

	if _, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil)); err != nil {
		t.Fatalf("request: %v", err)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 span, got %d", len(ended))
	}
	if got, want := ended[0].Name(), "GET /"; got != want {
		t.Fatalf("request span name = %q, want %q", got, want)
	}
}
