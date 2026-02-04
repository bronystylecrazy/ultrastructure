package web

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

type recordHandler struct {
	id    string
	order *[]string
}

func (h *recordHandler) Handle(r fiber.Router) {
	*h.order = append(*h.order, h.id)
}

func TestSetupHandlersPriorityOrder(t *testing.T) {
	app := fiber.New()
	var order []string

	h1 := &recordHandler{id: "h1", order: &order}
	h2 := &recordHandler{id: "h2", order: &order}
	h3 := &recordHandler{id: "h3", order: &order}
	h4 := &recordHandler{id: "h4", order: &order}

	fxApp := fx.New(di.App(
		di.Supply(app),
		di.Supply(h1, di.As[Handler](`group:"us.handlers"`)),
		di.Supply(h2, di.As[Handler](`group:"us.handlers"`), Priority(Later)),
		di.Supply(h3, di.As[Handler](`group:"us.handlers"`), Priority(Earlier)),
		di.Supply(h4, di.As[Handler](`group:"us.handlers"`), Priority(Later)),
		di.Invoke(SetupHandlers),
	).Build())
	if err := fxApp.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	if err := fxApp.Stop(context.Background()); err != nil {
		t.Fatalf("stop app: %v", err)
	}

	want := []string{"h3", "h1", "h2", "h4"}
	if len(order) != len(want) {
		t.Fatalf("order length mismatch: got %d want %d", len(order), len(want))
	}
	for i, got := range order {
		if got != want[i] {
			t.Fatalf("order[%d]=%q want %q", i, got, want[i])
		}
	}
}

func TestBetweenPriority(t *testing.T) {
	mid := Between(Earlier, Later)
	if mid != Normal {
		t.Fatalf("Between(Earlier, Later)=%v want %v", mid, Normal)
	}

	wide := Between(Latest, Earlier)
	if !(wide > Earlier && wide < Latest) {
		t.Fatalf("Between(Latest, Earlier)=%v want between %v and %v", wide, Earlier, Latest)
	}
}
