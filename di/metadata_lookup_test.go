package di

import "testing"

type lookupMeta struct {
	Name string
}

func TestFindMetadataReturnsFirstMatch(t *testing.T) {
	value := &struct{}{}
	RegisterMetadata(value, "first", lookupMeta{Name: "a"}, lookupMeta{Name: "b"})

	got, ok := FindMetadata[lookupMeta](value)
	if !ok {
		t.Fatalf("expected lookup metadata to be found")
	}
	if got.Name != "a" {
		t.Fatalf("unexpected first metadata: got %q want %q", got.Name, "a")
	}
}

func TestFindAllMetadataReturnsAllMatches(t *testing.T) {
	value := &struct{}{}
	RegisterMetadata(value, "first", lookupMeta{Name: "a"}, 123, lookupMeta{Name: "b"})

	got := FindAllMetadata[lookupMeta](value)
	if len(got) != 2 {
		t.Fatalf("unexpected metadata count: got %d want %d", len(got), 2)
	}
	if got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("unexpected metadata values: %#v", got)
	}
}

func TestFindMetadataReturnsFalseWhenMissing(t *testing.T) {
	value := &struct{}{}
	RegisterMetadata(value, "first", 123)

	_, ok := FindMetadata[lookupMeta](value)
	if ok {
		t.Fatalf("expected no lookup metadata match")
	}
}

func TestFindAllMetadataReturnsNilWhenMissing(t *testing.T) {
	value := &struct{}{}
	RegisterMetadata(value, "first", 123)

	got := FindAllMetadata[lookupMeta](value)
	if got != nil {
		t.Fatalf("expected nil when no matches, got %#v", got)
	}
}

