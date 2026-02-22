package session

import (
	"context"
	"time"
)

const DefaultRevocationKeyPrefix = "token:revoked:"
const DefaultRevocationNamespace = "default"

type RevocationStore interface {
	Revoke(ctx context.Context, jti string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type RevocationCache interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}
