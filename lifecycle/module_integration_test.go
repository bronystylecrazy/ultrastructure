package lifecycle

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ditest"
)

type lifecycleOrderRecorder struct {
	starts []string
	stops  []string
}

type starterEarly struct{ rec *lifecycleOrderRecorder }
type starterNormal struct{ rec *lifecycleOrderRecorder }
type starterLater struct{ rec *lifecycleOrderRecorder }

func (s *starterEarly) Start(context.Context) error {
	s.rec.starts = append(s.rec.starts, "early")
	return nil
}

func (s *starterNormal) Start(context.Context) error {
	s.rec.starts = append(s.rec.starts, "normal")
	return nil
}

func (s *starterLater) Start(context.Context) error {
	s.rec.starts = append(s.rec.starts, "later")
	return nil
}

type stopperEarly struct{ rec *lifecycleOrderRecorder }
type stopperNormal struct{ rec *lifecycleOrderRecorder }
type stopperLater struct{ rec *lifecycleOrderRecorder }

func (s *stopperEarly) Stop(context.Context) error {
	s.rec.stops = append(s.rec.stops, "early")
	return nil
}

func (s *stopperNormal) Stop(context.Context) error {
	s.rec.stops = append(s.rec.stops, "normal")
	return nil
}

func (s *stopperLater) Stop(context.Context) error {
	s.rec.stops = append(s.rec.stops, "later")
	return nil
}

func TestModuleStartPriorityOrdersStartersOnStart(t *testing.T) {
	rec := &lifecycleOrderRecorder{}
	app := ditest.New(t,
		Module(),
		di.Supply(rec),
		di.Provide(func(r *lifecycleOrderRecorder) *starterNormal { return &starterNormal{rec: r} }),
		di.Provide(func(r *lifecycleOrderRecorder) *starterLater { return &starterLater{rec: r} }, StartPriority(Later)),
		di.Provide(func(r *lifecycleOrderRecorder) *starterEarly { return &starterEarly{rec: r} }, StartPriority(Earlier)),
	)
	app.RequireStart()
	app.RequireStop()

	want := []string{"early", "normal", "later"}
	if len(rec.starts) != len(want) {
		t.Fatalf("unexpected start order length: got %d want %d (%v)", len(rec.starts), len(want), rec.starts)
	}
	for i := range want {
		if rec.starts[i] != want[i] {
			t.Fatalf("start order[%d]=%q want %q (%v)", i, rec.starts[i], want[i], rec.starts)
		}
	}
}

func TestModuleStopPriorityAffectsStopExecutionOrder(t *testing.T) {
	rec := &lifecycleOrderRecorder{}
	app := ditest.New(t,
		Module(),
		di.Supply(rec),
		di.Provide(func(r *lifecycleOrderRecorder) *stopperNormal { return &stopperNormal{rec: r} }),
		di.Provide(func(r *lifecycleOrderRecorder) *stopperLater { return &stopperLater{rec: r} }, StopPriority(Later)),
		di.Provide(func(r *lifecycleOrderRecorder) *stopperEarly { return &stopperEarly{rec: r} }, StopPriority(Earlier)),
	)
	app.RequireStart()
	app.RequireStop()

	// Fx executes OnStop hooks in reverse registration order (LIFO).
	// With registration sorted Early -> Normal -> Later, stop runs Later -> Normal -> Early.
	want := []string{"later", "normal", "early"}
	if len(rec.stops) != len(want) {
		t.Fatalf("unexpected stop order length: got %d want %d (%v)", len(rec.stops), len(want), rec.stops)
	}
	for i := range want {
		if rec.stops[i] != want[i] {
			t.Fatalf("stop order[%d]=%q want %q (%v)", i, rec.stops[i], want[i], rec.stops)
		}
	}
}
