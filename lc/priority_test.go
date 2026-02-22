package lc

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
)

type testStarter struct {
	id string
}

func (t *testStarter) Start(context.Context) error { return nil }

type testStopper struct {
	id string
}

func (t *testStopper) Stop(context.Context) error { return nil }

func TestSortStartersByPriorityStable(t *testing.T) {
	a := &testStarter{id: "a"}
	b := &testStarter{id: "b"}
	c := &testStarter{id: "c"}
	d := &testStarter{id: "d"}
	di.RegisterMetadata(b, startPriority{Index: int64(Earlier)})
	di.RegisterMetadata(c, startPriority{Index: int64(Earlier)})
	di.RegisterMetadata(d, startPriority{Index: int64(Later)})

	out := sortStartersByPriority([]Starter{a, b, c, d})
	got := []string{
		out[0].(*testStarter).id,
		out[1].(*testStarter).id,
		out[2].(*testStarter).id,
		out[3].(*testStarter).id,
	}
	want := []string{"b", "c", "a", "d"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("starter order[%d]=%q want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestSortStoppersByPriorityStable(t *testing.T) {
	a := &testStopper{id: "a"}
	b := &testStopper{id: "b"}
	c := &testStopper{id: "c"}
	d := &testStopper{id: "d"}
	di.RegisterMetadata(b, stopPriority{Index: int64(Earlier)})
	di.RegisterMetadata(c, stopPriority{Index: int64(Earlier)})
	di.RegisterMetadata(d, stopPriority{Index: int64(Later)})

	out := sortStoppersByPriority([]Stopper{a, b, c, d})
	got := []string{
		out[0].(*testStopper).id,
		out[1].(*testStopper).id,
		out[2].(*testStopper).id,
		out[3].(*testStopper).id,
	}
	want := []string{"b", "c", "a", "d"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stopper order[%d]=%q want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestStartPriorityLastMetadataWins(t *testing.T) {
	s := &testStarter{id: "s"}
	di.RegisterMetadata(s, startPriority{Index: int64(Earlier)}, startPriority{Index: int64(Latest)})
	if got := startPriorityIndex(s); got != int64(Latest) {
		t.Fatalf("unexpected start priority index: got %d want %d", got, int64(Latest))
	}
}

func TestStopPriorityLastMetadataWins(t *testing.T) {
	s := &testStopper{id: "s"}
	di.RegisterMetadata(s, stopPriority{Index: int64(Earlier)}, stopPriority{Index: int64(Latest)})
	if got := stopPriorityIndex(s); got != int64(Latest) {
		t.Fatalf("unexpected stop priority index: got %d want %d", got, int64(Latest))
	}
}
