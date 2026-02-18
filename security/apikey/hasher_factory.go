package apikey

import (
	"fmt"
	"strings"
)

const (
	HasherArgon2id   = "argon2id"
	HasherHMACSHA256 = "hmac-sha256"
)

func NewHasherFromConfig(cfg Config) (Hasher, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.HasherMode))
	switch mode {
	case "", HasherArgon2id:
		return NewArgon2idHasher(), nil
	case HasherHMACSHA256:
		return NewHMACSHA256Hasher(cfg.HMACSecret)
	default:
		return nil, fmt.Errorf("apikey: unsupported hasher mode: %s", cfg.HasherMode)
	}
}
