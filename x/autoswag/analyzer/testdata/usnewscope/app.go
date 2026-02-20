package usnewscope

import (
	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Responder interface {
	Ok(c fiber.Ctx) error
}

type handler struct {
	responder Responder
}

type responderA struct{}
type responderB struct{}

func NewResponderA() *responderA { return &responderA{} }
func NewResponderB() *responderB { return &responderB{} }

func (responderA) Ok(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}

func (responderB) Ok(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Error{})
}

func UseResponderA() di.Node {
	return di.Options(di.Provide(NewResponderA, di.As[Responder]()))
}

func UseResponderB() di.Node {
	return di.Options(di.Provide(NewResponderB, di.As[Responder]()))
}

func UseApp() {
	us.New(
		UseResponderA(),
		// Intentionally not wiring UseResponderB().
	)
}

func (h *handler) Handle(r web.Router) {
	r.Get("/scoped", h.Wrapped)
}

func (h *handler) Wrapped(c fiber.Ctx) error {
	return h.responder.Ok(c)
}
