package us_test

import "testing"

import us "github.com/bronystylecrazy/ultrastructure"
import "github.com/bronystylecrazy/ultrastructure/meta"

func TestBuildModeHelpers(t *testing.T) {
	originalVersion := meta.Version
	defer func() { meta.Version = originalVersion }()

	meta.Version = us.NilVersion
	if !us.IsDevelopment() {
		t.Fatal("expected development mode when version is nil")
	}
	if us.IsProduction() {
		t.Fatal("did not expect production mode when version is nil")
	}

	meta.Version = "1.2.3"
	if us.IsDevelopment() {
		t.Fatal("did not expect development mode when version is set")
	}
	if !us.IsProduction() {
		t.Fatal("expected production mode when version is set")
	}
}
