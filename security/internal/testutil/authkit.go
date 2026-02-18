package testutil

import (
	"context"

	apikey "github.com/bronystylecrazy/ultrastructure/security/apikey"
	token "github.com/bronystylecrazy/ultrastructure/security/token"
)

type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

type memoryLookup struct {
	data map[string]*apikey.StoredKey
}

func (m memoryLookup) FindByKeyID(ctx context.Context, keyID string) (*apikey.StoredKey, error) {
	return m.data[keyID], nil
}

func NewAPIKeyManager(tb TB) (apikey.Manager, string) {
	tb.Helper()

	cfg := apikey.Config{}
	gen := apikey.NewKeyGenerator(cfg)
	hasher := apikey.NewArgon2idHasher()

	raw, keyID, secret, err := gen.GenerateRawKey("intg")
	if err != nil {
		tb.Fatalf("GenerateRawKey: %v", err)
	}
	hash, err := hasher.Hash(secret)
	if err != nil {
		tb.Fatalf("Hash: %v", err)
	}
	lookup := memoryLookup{
		data: map[string]*apikey.StoredKey{
			keyID: {
				KeyID:      keyID,
				AppID:      "app-1",
				SecretHash: hash,
				Scopes:     []string{"read:orders"},
			},
		},
	}
	m := apikey.NewService(apikey.NewServiceParams{
		Config:    cfg,
		Generator: gen,
		Hasher:    hasher,
		Lookup:    lookup,
	})
	return m, raw
}

func NewUserManager(tb TB) (token.Manager, string) {
	tb.Helper()

	m, err := token.NewService(token.Config{Secret: "test-secret"})
	if err != nil {
		tb.Fatalf("NewService: %v", err)
	}
	pair, err := m.GenerateTokenPair("user-1", map[string]any{
		"role":  "admin",
		"scope": "read:orders write:orders",
	})
	if err != nil {
		tb.Fatalf("GenerateTokenPair: %v", err)
	}
	return m, pair.AccessToken
}
