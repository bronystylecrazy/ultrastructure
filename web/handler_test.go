package web

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ditest"
	"github.com/gofiber/fiber/v3"
)

type recordHandler struct {
	id    string
	order *[]string
}

func (h *recordHandler) append() {
	*h.order = append(*h.order, h.id)
}

type recordHandler1 struct{ recordHandler }
type recordHandler2 struct{ recordHandler }
type recordHandler3 struct{ recordHandler }
type recordHandler4 struct{ recordHandler }

func (h *recordHandler1) Handle(r Router) { h.append() }
func (h *recordHandler2) Handle(r Router) { h.append() }
func (h *recordHandler3) Handle(r Router) { h.append() }
func (h *recordHandler4) Handle(r Router) { h.append() }

func TestSetupHandlersPriorityOrder(t *testing.T) {
	app := fiber.New()
	var order []string

	h1 := &recordHandler1{recordHandler{id: "h1", order: &order}}
	h2 := &recordHandler2{recordHandler{id: "h2", order: &order}}
	h3 := &recordHandler3{recordHandler{id: "h3", order: &order}}
	h4 := &recordHandler4{recordHandler{id: "h4", order: &order}}

	fxApp := ditest.New(t,
		di.AutoGroup[Handler](HandlersGroupName),
		di.Supply(app),
		di.Provide(NewRegistryContainer),
		di.Supply(h1),
		di.Supply(h2, Priority(Later)),
		di.Supply(h3, Priority(Earlier)),
		di.Supply(h4, Priority(Later)),
		di.Invoke(SetupHandlers),
	)
	defer fxApp.RequireStart().RequireStop()

	if len(order) != 4 {
		t.Fatalf("order length mismatch: got %d want %d", len(order), 4)
	}
	if order[0] != "h3" {
		t.Fatalf("expected earliest handler first, got %v", order)
	}
	if order[1] != "h1" {
		t.Fatalf("expected normal handler second, got %v", order)
	}
	last := map[string]bool{order[2]: true, order[3]: true}
	if !last["h2"] || !last["h4"] {
		t.Fatalf("expected later handlers at the end, got %v", order)
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

func TestSetupHandlersWebPriorityIsIndependentFromDIPriority(t *testing.T) {
	app := fiber.New()
	var order []string

	h1 := &recordHandler1{recordHandler{id: "h1", order: &order}}
	h2 := &recordHandler2{recordHandler{id: "h2", order: &order}}
	h3 := &recordHandler3{recordHandler{id: "h3", order: &order}}

	fxApp := ditest.New(t,
		di.AutoGroup[Handler](HandlersGroupName),
		di.Supply(app),
		di.Provide(NewRegistryContainer),
		di.Supply(h1, di.Priority(di.Earliest)),
		di.Supply(h2, Priority(Earlier), di.Priority(di.Latest)),
		di.Supply(h3),
		di.Invoke(SetupHandlers),
	)
	defer fxApp.RequireStart().RequireStop()

	if len(order) != 3 {
		t.Fatalf("order length mismatch: got %d want %d", len(order), 3)
	}
	if order[0] != "h2" {
		t.Fatalf("expected web-priority handler first, got %v", order)
	}
	last := map[string]bool{order[1]: true, order[2]: true}
	if !last["h1"] || !last["h3"] {
		t.Fatalf("expected remaining handlers to keep non-web priority bucket, got %v", order)
	}
}

func TestSetupHandlersWebPriorityLastWins(t *testing.T) {
	app := fiber.New()
	var order []string

	h1 := &recordHandler1{recordHandler{id: "h1", order: &order}}
	h2 := &recordHandler2{recordHandler{id: "h2", order: &order}}

	fxApp := ditest.New(t,
		di.AutoGroup[Handler](HandlersGroupName),
		di.Supply(app),
		di.Provide(NewRegistryContainer),
		di.Supply(h1, Priority(Earlier), Priority(Latest)),
		di.Supply(h2),
		di.Invoke(SetupHandlers),
	)
	defer fxApp.RequireStart().RequireStop()

	want := []string{"h2", "h1"}
	if len(order) != len(want) {
		t.Fatalf("order length mismatch: got %d want %d", len(order), len(want))
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d]=%q want %q (%v)", i, order[i], want[i], order)
		}
	}
}
