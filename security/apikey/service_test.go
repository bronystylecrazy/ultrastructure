package apikey

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

type inMemoryLookup struct {
	data map[string]*StoredKey
}

func (m inMemoryLookup) FindByKeyID(ctx context.Context, keyID string) (*StoredKey, error) {
	v, ok := m.data[keyID]
	if !ok {
		return nil, nil
	}
	return v, nil
}

type usageRecorder struct {
	lastKeyID string
}

func (u *usageRecorder) MarkUsed(ctx context.Context, keyID string, usedAt time.Time) error {
	u.lastKeyID = keyID
	return nil
}

func TestIssueKeyAndParse(t *testing.T) {
	cfg := Config{}.withDefaults()
	gen := NewKeyGenerator(cfg)
	hasher := NewArgon2idHasher()
	svc := NewService(NewServiceParams{
		Config:    cfg,
		Generator: gen,
		Hasher:    hasher,
	})

	issued, err := svc.IssueKey("app-1", "intg", []string{"read:orders"}, map[string]string{"env": "test"}, nil)
	if err != nil {
		t.Fatalf("IssueKey: %v", err)
	}
	if issued.RawKey == "" || issued.KeyID == "" || issued.SecretHash == "" {
		t.Fatal("expected non-empty issued key fields")
	}
	keyID, secret, err := gen.ParseRawKey(issued.RawKey)
	if err != nil {
		t.Fatalf("ParseRawKey: %v", err)
	}
	if keyID != issued.KeyID || secret == "" {
		t.Fatalf("parse mismatch: keyID=%q issued=%q secretEmpty=%v", keyID, issued.KeyID, secret == "")
	}
}

func TestArgon2idHasher(t *testing.T) {
	h := NewArgon2idHasher()
	hash, err := h.Hash("secret-123")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	ok, err := h.Verify(hash, "secret-123")
	if err != nil {
		t.Fatalf("Verify(good): %v", err)
	}
	if !ok {
		t.Fatal("expected hash verification success")
	}
	ok, err = h.Verify(hash, "secret-xyz")
	if err != nil {
		t.Fatalf("Verify(bad): %v", err)
	}
	if ok {
		t.Fatal("expected hash verification failure")
	}
}

func TestMiddlewareSetsPrincipal(t *testing.T) {
	cfg := Config{}.withDefaults()
	cfg.SetPrincipalBody = true
	gen := NewKeyGenerator(cfg)
	hasher := NewArgon2idHasher()
	rec := &usageRecorder{}
	raw, keyID, secret, err := gen.GenerateRawKey("intg")
	if err != nil {
		t.Fatalf("GenerateRawKey: %v", err)
	}
	hash, err := hasher.Hash(secret)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	lookup := inMemoryLookup{
		data: map[string]*StoredKey{
			keyID: {
				KeyID:      keyID,
				AppID:      "app-1",
				SecretHash: hash,
				Scopes:     []string{"read:orders"},
			},
		},
	}
	svc := NewService(NewServiceParams{
		Config:    cfg,
		Generator: gen,
		Hasher:    hasher,
		Lookup:    lookup,
		Recorder:  rec,
	})

	app := fiber.New()
	app.Get("/p", svc.Middleware(), func(c fiber.Ctx) error {
		p1, ok := PrincipalFromLocals(c)
		if !ok || p1 == nil {
			return c.Status(fiber.StatusInternalServerError).SendString("missing locals principal")
		}
		p2, ok := PrincipalFromContext(c.Context())
		if !ok || p2 == nil {
			return c.Status(fiber.StatusInternalServerError).SendString("missing ctx principal")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "ApiKey "+raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
	if rec.lastKeyID != keyID {
		t.Fatalf("usage recorder mismatch: got=%q want=%q", rec.lastKeyID, keyID)
	}
}

func TestMiddlewareGenericErrorByDefault(t *testing.T) {
	cfg := Config{}.withDefaults()
	gen := NewKeyGenerator(cfg)
	hasher := NewArgon2idHasher()
	raw, keyID, secret, err := gen.GenerateRawKey("intg")
	if err != nil {
		t.Fatalf("GenerateRawKey: %v", err)
	}
	hash, err := hasher.Hash(secret)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	now := time.Now().UTC()
	lookup := inMemoryLookup{
		data: map[string]*StoredKey{
			keyID: {
				KeyID:      keyID,
				AppID:      "app-1",
				SecretHash: hash,
				RevokedAt:  &now,
			},
		},
	}
	svc := NewService(NewServiceParams{
		Config:    cfg, // DetailedErrors=false default
		Generator: gen,
		Hasher:    hasher,
		Lookup:    lookup,
	})

	app := fiber.New()
	app.Get("/p", svc.Middleware(), func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "ApiKey "+raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusUnauthorized)
	}
	body := make([]byte, 512)
	n, _ := res.Body.Read(body)
	if !strings.Contains(string(body[:n]), ErrInvalidAPIKey.Error()) {
		t.Fatalf("expected generic error body, got=%q", string(body[:n]))
	}
}

func TestRevokeRotateNotConfigured(t *testing.T) {
	cfg := Config{}.withDefaults()
	svc := NewService(NewServiceParams{
		Config:    cfg,
		Generator: NewKeyGenerator(cfg),
		Hasher:    NewArgon2idHasher(),
	})
	if err := svc.RevokeKey(context.Background(), "kid", "logout"); !errors.Is(err, ErrRevokerNotConfigured) {
		t.Fatalf("RevokeKey err: got=%v want=%v", err, ErrRevokerNotConfigured)
	}
	if _, err := svc.RotateKey(context.Background(), "kid", "pfx"); !errors.Is(err, ErrRotatorNotConfigured) {
		t.Fatalf("RotateKey err: got=%v want=%v", err, ErrRotatorNotConfigured)
	}
}
