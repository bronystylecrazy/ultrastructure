package apikey

import (
	"context"
	"testing"
	"time"
)

type countingLookup struct {
	count int
	data  map[string]*StoredKey
}

func (c *countingLookup) FindByKeyID(ctx context.Context, keyID string) (*StoredKey, error) {
	c.count++
	return cloneStoredKey(c.data[keyID]), nil
}

type memoryRevoker struct {
	data map[string]*StoredKey
}

func (m *memoryRevoker) RevokeKey(ctx context.Context, keyID string, reason string) error {
	v := m.data[keyID]
	if v == nil {
		return nil
	}
	now := time.Now().UTC()
	v.RevokedAt = &now
	return nil
}

func TestCachedLookupAvoidsRepeatedBackendCalls(t *testing.T) {
	exp := time.Now().UTC().Add(10 * time.Minute)
	base := &countingLookup{
		data: map[string]*StoredKey{
			"k1": {
				KeyID:      "k1",
				AppID:      "app-1",
				SecretHash: "hash",
				ExpiresAt:  &exp,
			},
		},
	}
	cache := NewCachedLookup(base, nil, CachedLookupConfig{
		L1TTL: 5 * time.Minute,
	})

	got1, err := cache.FindByKeyID(context.Background(), "k1")
	if err != nil || got1 == nil {
		t.Fatalf("first lookup: key=%v err=%v", got1, err)
	}
	got2, err := cache.FindByKeyID(context.Background(), "k1")
	if err != nil || got2 == nil {
		t.Fatalf("second lookup: key=%v err=%v", got2, err)
	}
	if base.count != 1 {
		t.Fatalf("backend calls: got=%d want=%d", base.count, 1)
	}
}

func TestCachedLookupNegativeCaching(t *testing.T) {
	base := &countingLookup{
		data: map[string]*StoredKey{},
	}
	cache := NewCachedLookup(base, nil, CachedLookupConfig{
		L1TTL:       5 * time.Minute,
		NegativeTTL: 2 * time.Minute,
	})

	got1, err := cache.FindByKeyID(context.Background(), "missing")
	if err != nil || got1 != nil {
		t.Fatalf("first missing lookup: key=%v err=%v", got1, err)
	}
	got2, err := cache.FindByKeyID(context.Background(), "missing")
	if err != nil || got2 != nil {
		t.Fatalf("second missing lookup: key=%v err=%v", got2, err)
	}
	if base.count != 1 {
		t.Fatalf("backend calls for missing key: got=%d want=%d", base.count, 1)
	}
}

func TestCachedRevokerInvalidatesLookup(t *testing.T) {
	exp := time.Now().UTC().Add(10 * time.Minute)
	source := map[string]*StoredKey{
		"k1": {
			KeyID:      "k1",
			AppID:      "app-1",
			SecretHash: "hash",
			ExpiresAt:  &exp,
		},
	}
	base := &countingLookup{data: source}
	cache := NewCachedLookup(base, nil, CachedLookupConfig{
		L1TTL: 5 * time.Minute,
	})
	revoker := NewCachedRevoker(&memoryRevoker{data: source}, cache)

	keyBefore, err := cache.FindByKeyID(context.Background(), "k1")
	if err != nil || keyBefore == nil || keyBefore.RevokedAt != nil {
		t.Fatalf("lookup before revoke: key=%v err=%v", keyBefore, err)
	}
	if err := revoker.RevokeKey(context.Background(), "k1", "test"); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	keyAfter, err := cache.FindByKeyID(context.Background(), "k1")
	if err != nil || keyAfter == nil || keyAfter.RevokedAt == nil {
		t.Fatalf("lookup after revoke: key=%v err=%v", keyAfter, err)
	}
	if base.count != 2 {
		t.Fatalf("backend calls after invalidate: got=%d want=%d", base.count, 2)
	}
}
