package imgutil_test

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/imgutil"
	"go.uber.org/fx/fxtest"
)

func TestModuleProvidesServiceAndInterface(t *testing.T) {
	var svc *imgutil.Service
	var hasher imgutil.ThumbHasher

	defer fxtest.New(
		t,
		di.App(
			imgutil.Module(),
			di.Populate(&svc),
			di.Populate(&hasher),
		).Build(),
	).RequireStart().RequireStop()

	if svc == nil {
		t.Fatal("service is nil")
	}
	if hasher == nil {
		t.Fatal("thumb hasher is nil")
	}
}
