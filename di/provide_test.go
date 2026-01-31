package di

import (
	"reflect"
	"testing"
)

type provideIface interface {
	ID() string
}

type provideOtherIface interface {
	Other() string
}

type provideImpl struct{}

func (p *provideImpl) ID() string    { return "id" }
func (p *provideImpl) Other() string { return "other" }

func newProvideImpl() *provideImpl { return &provideImpl{} }

func TestBuildProvideSpecAutoGroupAddsGroupAndSelf(t *testing.T) {
	iface := reflect.TypeOf((*provideIface)(nil)).Elem()
	cfg := bindConfig{
		autoGroups: []autoGroupRule{{iface: iface, group: "grp"}},
	}
	spec, tagSets, err := buildProvideSpec(cfg, newProvideImpl, nil)
	if err != nil {
		t.Fatalf("buildProvideSpec: %v", err)
	}
	if !spec.includeSelf {
		t.Fatalf("expected includeSelf true")
	}
	if !hasTagSet(tagSets, tagSet{typ: iface, group: "grp"}) {
		t.Fatalf("expected group tag set, got %#v", tagSets)
	}
}

func TestBuildProvideSpecAutoGroupIgnoreType(t *testing.T) {
	iface := reflect.TypeOf((*provideIface)(nil)).Elem()
	cfg := bindConfig{
		autoGroups:       []autoGroupRule{{iface: iface, group: "grp"}},
		autoGroupIgnores: []autoGroupRule{{iface: iface, group: "grp"}},
	}
	_, tagSets, err := buildProvideSpec(cfg, newProvideImpl, nil)
	if err != nil {
		t.Fatalf("buildProvideSpec: %v", err)
	}
	if hasTagSet(tagSets, tagSet{typ: iface, group: "grp"}) {
		t.Fatalf("expected group tag set to be ignored, got %#v", tagSets)
	}
}

func TestBuildProvideSpecAutoGroupFilter(t *testing.T) {
	iface := reflect.TypeOf((*provideIface)(nil)).Elem()
	cfg := bindConfig{
		autoGroups: []autoGroupRule{{
			iface:  iface,
			group:  "grp",
			filter: func(reflect.Type) bool { return false },
		}},
	}
	_, tagSets, err := buildProvideSpec(cfg, newProvideImpl, nil)
	if err != nil {
		t.Fatalf("buildProvideSpec: %v", err)
	}
	if hasTagSet(tagSets, tagSet{typ: iface, group: "grp"}) {
		t.Fatalf("expected filtered group tag set to be absent, got %#v", tagSets)
	}
}

func TestBuildProvideSpecAutoGroupAsSelfWithExplicitExport(t *testing.T) {
	iface := reflect.TypeOf((*provideIface)(nil)).Elem()
	otherIface := reflect.TypeOf((*provideOtherIface)(nil)).Elem()
	cfg := bindConfig{
		exports: []exportSpec{{typ: otherIface}},
		autoGroups: []autoGroupRule{{
			iface:  iface,
			group:  "grp",
			asSelf: true,
		}},
	}
	spec, tagSets, err := buildProvideSpec(cfg, newProvideImpl, nil)
	if err != nil {
		t.Fatalf("buildProvideSpec: %v", err)
	}
	if !spec.includeSelf {
		t.Fatalf("expected includeSelf true with AutoGroupAsSelf")
	}
	if !hasTagSet(tagSets, tagSet{typ: iface, group: "grp"}) {
		t.Fatalf("expected group tag set, got %#v", tagSets)
	}
	if !hasTagSet(tagSets, tagSet{typ: otherIface}) {
		t.Fatalf("expected explicit export tag set, got %#v", tagSets)
	}
}

type providePtrIface interface {
	Ping() string
}

type providePtrImpl struct{}

func (p *providePtrImpl) Ping() string { return "pong" }

func newProvidePtrImpl() providePtrImpl { return providePtrImpl{} }

func TestBuildProvideSpecAutoGroupPointerReceiver(t *testing.T) {
	iface := reflect.TypeOf((*providePtrIface)(nil)).Elem()
	cfg := bindConfig{
		autoGroups: []autoGroupRule{{
			iface:  iface,
			group:  "ptrs",
			filter: func(reflect.Type) bool { return true },
			asSelf: true,
		}},
	}
	spec, tagSets, err := buildProvideSpec(cfg, newProvidePtrImpl, nil)
	if err != nil {
		t.Fatalf("buildProvideSpec: %v", err)
	}
	if !spec.includeSelf {
		t.Fatalf("expected includeSelf true")
	}
	if !hasTagSet(tagSets, tagSet{typ: iface, group: "ptrs"}) {
		t.Fatalf("expected group tag set, got %#v", tagSets)
	}
}

func hasTagSet(tagSets []tagSet, needle tagSet) bool {
	for _, ts := range tagSets {
		if ts.typ != needle.typ {
			continue
		}
		if ts.name != needle.name {
			continue
		}
		if ts.group != needle.group {
			continue
		}
		return true
	}
	return false
}

func hasTagSetType(tagSets []tagSet, typ reflect.Type) bool {
	for _, ts := range tagSets {
		if ts.typ == typ && ts.name == "" && ts.group == "" {
			return true
		}
	}
	return false
}
