package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHashFormat = errors.New("apikey: invalid hash format")
)

// Hash format: v1$base64(salt)$base64(sha256(salt || secret))
type SaltedSHA256Hasher struct{}

func NewSaltedSHA256Hasher() *SaltedSHA256Hasher {
	return &SaltedSHA256Hasher{}
}

func (h *SaltedSHA256Hasher) Hash(secret string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(append([]byte{}, salt...), []byte(secret)...))
	return fmt.Sprintf("v1$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum[:]),
	), nil
}

func (h *SaltedSHA256Hasher) Verify(hash string, secret string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 3 || parts[0] != "v1" {
		return false, ErrInvalidHashFormat
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	sum := sha256.Sum256(append(append([]byte{}, salt...), []byte(secret)...))
	ok := subtle.ConstantTimeCompare(want, sum[:]) == 1
	return ok, nil
}

type HMACSHA256Hasher struct {
	secret []byte
}

func NewHMACSHA256Hasher(secret string) (*HMACSHA256Hasher, error) {
	s := strings.TrimSpace(secret)
	if s == "" {
		return nil, errors.New("apikey: hmac secret is required")
	}
	return &HMACSHA256Hasher{secret: []byte(s)}, nil
}

func (h *HMACSHA256Hasher) Hash(secret string) (string, error) {
	m := hmac.New(sha256.New, h.secret)
	_, _ = m.Write([]byte(secret))
	return "h1$" + base64.RawStdEncoding.EncodeToString(m.Sum(nil)), nil
}

func (h *HMACSHA256Hasher) Verify(hash string, secret string) (bool, error) {
	if !strings.HasPrefix(hash, "h1$") {
		return false, ErrInvalidHashFormat
	}
	want, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(hash, "h1$"))
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	m := hmac.New(sha256.New, h.secret)
	_, _ = m.Write([]byte(secret))
	got := m.Sum(nil)
	return subtle.ConstantTimeCompare(want, got) == 1, nil
}

// Hash format: a1$memory$time$threads$base64(salt)$base64(hash)
type Argon2idHasher struct {
	memoryKB uint32
	timeCost uint32
	threads  uint8
	keyLen   uint32
}

func NewArgon2idHasher() *Argon2idHasher {
	return &Argon2idHasher{
		memoryKB: 64 * 1024,
		timeCost: 3,
		threads:  2,
		keyLen:   32,
	}
}

func (h *Argon2idHasher) Hash(secret string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	out := argon2.IDKey([]byte(secret), salt, h.timeCost, h.memoryKB, h.threads, h.keyLen)
	return fmt.Sprintf("a1$%d$%d$%d$%s$%s",
		h.memoryKB,
		h.timeCost,
		h.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(out),
	), nil
}

func (h *Argon2idHasher) Verify(hash string, secret string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "a1" {
		return false, ErrInvalidHashFormat
	}
	memoryKB, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	timeCost, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	threads, err := strconv.ParseUint(parts[3], 10, 8)
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	got := argon2.IDKey([]byte(secret), salt, uint32(timeCost), uint32(memoryKB), uint8(threads), uint32(len(want)))
	return subtle.ConstantTimeCompare(want, got) == 1, nil
}
