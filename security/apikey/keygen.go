package apikey

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidRawKeyFormat = errors.New("apikey: invalid raw key format")
)

type KeyGenerator struct {
	idLength     int
	secretLength int
}

func NewKeyGenerator(cfg Config) *KeyGenerator {
	c := cfg.withDefaults()
	return &KeyGenerator{
		idLength:     c.KeyIDLength,
		secretLength: c.SecretLength,
	}
}

func (g *KeyGenerator) GenerateRawKey(prefix string) (rawKey string, keyID string, secret string, err error) {
	pfx := strings.TrimSpace(prefix)
	if pfx == "" {
		pfx = defaultKeyPrefix
	}
	id, err := randomBase32(g.idLength)
	if err != nil {
		return "", "", "", err
	}
	sec, err := randomBase32(g.secretLength)
	if err != nil {
		return "", "", "", err
	}
	return fmt.Sprintf("%s_%s.%s", pfx, id, sec), id, sec, nil
}

func (g *KeyGenerator) ParseRawKey(rawKey string) (keyID string, secret string, err error) {
	value := strings.TrimSpace(rawKey)
	p := strings.SplitN(value, ".", 2)
	if len(p) != 2 || p[1] == "" {
		return "", "", ErrInvalidRawKeyFormat
	}
	left := p[0]
	idx := strings.LastIndex(left, "_")
	if idx <= 0 || idx >= len(left)-1 {
		return "", "", ErrInvalidRawKeyFormat
	}
	keyID = left[idx+1:]
	secret = p[1]
	if keyID == "" || secret == "" {
		return "", "", ErrInvalidRawKeyFormat
	}
	return keyID, secret, nil
}

func randomBase32(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	v := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	if len(v) > length {
		v = v[:length]
	}
	return strings.ToLower(v), nil
}
