package license

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrClockRollback      = errors.New("clock rollback detected")
	ErrClockStateTampered = errors.New("clock state tampered")
)

type FileTimeGuard struct {
	path      string
	tolerance time.Duration
	macKey    []byte
}

type timeGuardState struct {
	LastVerifiedUnix int64  `json:"last_verified_unix"`
	MAC              string `json:"mac,omitempty"`
}

func NewFileTimeGuard(path string, tolerance time.Duration) *FileTimeGuard {
	return &FileTimeGuard{
		path:      path,
		tolerance: tolerance,
	}
}

func NewFileTimeGuardWithMAC(path string, tolerance time.Duration, macKey []byte) *FileTimeGuard {
	keyCopy := make([]byte, len(macKey))
	copy(keyCopy, macKey)
	return &FileTimeGuard{
		path:      path,
		tolerance: tolerance,
		macKey:    keyCopy,
	}
}

func (g *FileTimeGuard) CheckAndUpdate(ctx context.Context, now time.Time) error {
	if g == nil {
		return errors.New("nil time guard")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	nowUnix := now.UTC().Unix()
	state, err := g.readState()
	if err != nil {
		return err
	}

	// Reject meaningful backward jumps. A small tolerance helps with normal clock drift.
	if state.LastVerifiedUnix > 0 && nowUnix+int64(g.tolerance.Seconds()) < state.LastVerifiedUnix {
		return fmt.Errorf("%w: now=%d last=%d tolerance=%s", ErrClockRollback, nowUnix, state.LastVerifiedUnix, g.tolerance.String())
	}

	if nowUnix > state.LastVerifiedUnix {
		state.LastVerifiedUnix = nowUnix
		if err := g.writeState(state); err != nil {
			return err
		}
	}

	return nil
}

func (g *FileTimeGuard) readState() (timeGuardState, error) {
	raw, err := os.ReadFile(g.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return timeGuardState{}, nil
		}
		return timeGuardState{}, err
	}

	var state timeGuardState
	if err := json.Unmarshal(raw, &state); err != nil {
		return timeGuardState{}, err
	}
	if len(g.macKey) > 0 {
		if state.MAC == "" {
			return timeGuardState{}, fmt.Errorf("%w: missing state mac", ErrClockStateTampered)
		}
		if !hmac.Equal([]byte(state.MAC), []byte(g.signMAC(state.LastVerifiedUnix))) {
			return timeGuardState{}, fmt.Errorf("%w: invalid state mac", ErrClockStateTampered)
		}
	}
	return state, nil
}

func (g *FileTimeGuard) writeState(state timeGuardState) error {
	if err := os.MkdirAll(filepath.Dir(g.path), 0o755); err != nil {
		return err
	}
	if len(g.macKey) > 0 {
		state.MAC = g.signMAC(state.LastVerifiedUnix)
	} else {
		state.MAC = ""
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}

	tmp := g.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, g.path)
}

func (g *FileTimeGuard) signMAC(lastVerifiedUnix int64) string {
	mac := hmac.New(sha256.New, g.macKey)
	_, _ = mac.Write([]byte(fmt.Sprintf("%d", lastVerifiedUnix)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
