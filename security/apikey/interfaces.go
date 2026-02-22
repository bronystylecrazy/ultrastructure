package apikey

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

type StoredKey struct {
	KeyID      string
	AppID      string
	SecretHash string
	Scopes     []string
	Metadata   map[string]string
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
}

type IssuedKey struct {
	KeyID      string            `json:"key_id"`
	AppID      string            `json:"app_id"`
	RawKey     string            `json:"raw_key"`
	Prefix     string            `json:"prefix"`
	SecretHash string            `json:"-"`
	Scopes     []string          `json:"scopes"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
}

type Generator interface {
	GenerateRawKey(prefix string) (rawKey string, keyID string, secret string, err error)
	ParseRawKey(rawKey string) (keyID string, secret string, err error)
}

type Hasher interface {
	Hash(secret string) (string, error)
	Verify(hash string, secret string) (bool, error)
}

type KeyLookup interface {
	FindByKeyID(ctx context.Context, keyID string) (*StoredKey, error)
}

type KeyUsageRecorder interface {
	MarkUsed(ctx context.Context, keyID string, usedAt time.Time) error
}

type Revoker interface {
	RevokeKey(ctx context.Context, keyID string, reason string) error
}

type Rotator interface {
	RotateKey(ctx context.Context, keyID string, prefix string) (*IssuedKey, error)
}

type Manager interface {
	IssueKey(appID string, opts ...IssueOption) (*IssuedKey, error)
	ValidateRawKey(ctx context.Context, rawKey string) (*Principal, error)
	Middleware() fiber.Handler
	RevokeKey(ctx context.Context, keyID string, reason string) error
	RotateKey(ctx context.Context, keyID string, prefix string) (*IssuedKey, error)
}
