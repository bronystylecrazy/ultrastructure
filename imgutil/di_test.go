package imgutil_test

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ditest"
	"github.com/bronystylecrazy/ultrastructure/imgutil"
)

func TestModuleProvidesServiceAndInterface(t *testing.T) {
	var svc *imgutil.Service
	var hasher imgutil.ThumbHasher

	defer ditest.New(
		t,
		imgutil.Providers(),
		di.Populate(&svc),
		di.Populate(&hasher),
	).RequireStart().RequireStop()

	if svc == nil {
		t.Fatal("service is nil")
	}
	if hasher == nil {
		t.Fatal("thumb hasher is nil")
	}
}
