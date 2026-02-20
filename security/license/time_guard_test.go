package license

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileTimeGuard_CheckAndUpdate_InitialWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clock_state.json")
	guard := NewFileTimeGuard(path, 2*time.Minute)

	now := time.Unix(1_700_000_000, 0).UTC()
	if err := guard.CheckAndUpdate(context.Background(), now); err != nil {
		t.Fatalf("CheckAndUpdate: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected state file to be written")
	}
}

func TestFileTimeGuard_CheckAndUpdate_RejectsRollback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clock_state.json")
	guard := NewFileTimeGuard(path, time.Minute)

	first := time.Unix(1_700_000_000, 0).UTC()
	if err := guard.CheckAndUpdate(context.Background(), first); err != nil {
		t.Fatalf("first CheckAndUpdate: %v", err)
	}

	rollback := first.Add(-2 * time.Minute)
	err := guard.CheckAndUpdate(context.Background(), rollback)
	if !errors.Is(err, ErrClockRollback) {
		t.Fatalf("expected ErrClockRollback, got: %v", err)
	}
}

func TestFileTimeGuard_CheckAndUpdate_AllowsSmallDrift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clock_state.json")
	guard := NewFileTimeGuard(path, 2*time.Minute)

	first := time.Unix(1_700_000_000, 0).UTC()
	if err := guard.CheckAndUpdate(context.Background(), first); err != nil {
		t.Fatalf("first CheckAndUpdate: %v", err)
	}

	// 60s backward is within 120s tolerance.
	drift := first.Add(-60 * time.Second)
	if err := guard.CheckAndUpdate(context.Background(), drift); err != nil {
		t.Fatalf("expected tolerated drift, got: %v", err)
	}
}

func TestFileTimeGuard_CheckAndUpdate_ContextCanceled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clock_state.json")
	guard := NewFileTimeGuard(path, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := guard.CheckAndUpdate(ctx, time.Now())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestFileTimeGuardWithMAC_CheckAndUpdate_OK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clock_state.json")
	guard := NewFileTimeGuardWithMAC(path, time.Minute, []byte("test-mac-key"))

	now := time.Unix(1_700_000_000, 0).UTC()
	if err := guard.CheckAndUpdate(context.Background(), now); err != nil {
		t.Fatalf("CheckAndUpdate: %v", err)
	}

	if err := guard.CheckAndUpdate(context.Background(), now.Add(time.Minute)); err != nil {
		t.Fatalf("CheckAndUpdate second call: %v", err)
	}
}

func TestFileTimeGuardWithMAC_CheckAndUpdate_TamperedState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clock_state.json")
	guard := NewFileTimeGuardWithMAC(path, time.Minute, []byte("test-mac-key"))

	now := time.Unix(1_700_000_000, 0).UTC()
	if err := guard.CheckAndUpdate(context.Background(), now); err != nil {
		t.Fatalf("first CheckAndUpdate: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}

	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	state["last_verified_unix"] = float64(now.Add(10 * time.Hour).Unix())
	tampered, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal tampered state: %v", err)
	}
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatalf("write tampered state: %v", err)
	}

	err = guard.CheckAndUpdate(context.Background(), now.Add(time.Minute))
	if !errors.Is(err, ErrClockStateTampered) {
		t.Fatalf("expected ErrClockStateTampered, got: %v", err)
	}
}
