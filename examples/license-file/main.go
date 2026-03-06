package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	licensepkg "github.com/bronystylecrazy/ultrastructure/security/license"
)

// Embedded in binary only (demo key).
// Replace with your real secret bytes generation flow.
const embeddedClockGuardKeyB64 = "dWx0cmFzdHJ1Y3R1cmUtZGVtby1jbG9jay1ndWFyZC1rZXk"

func main() {
	base := "examples/license-file/testdata"
	publicKeysPath := filepath.Join(base, "public_keys.json")
	licensePath := chooseLicensePath(base)
	clockStatePath := filepath.Join(base, "clock_state_signed.json")

	pubKeys, err := loadPublicKeys(publicKeysPath)
	if err != nil {
		log.Fatalf("load public keys: %v", err)
	}
	verifier, err := licensepkg.NewVerifier(
		licensepkg.WithPublicKeyProvider(licensepkg.StaticPublicKeyProvider(pubKeys)),
		licensepkg.WithHardwareDetector(licensepkg.NewHardwareDetector()),
	)
	if err != nil {
		log.Fatalf("new verifier: %v", err)
	}

	tokenBytes, err := os.ReadFile(licensePath)
	if err != nil {
		log.Fatalf("read license file: %v", err)
	}
	token := string(tokenBytes)
	fmt.Printf("using license file: %s\n", licensePath)

	expectedDevice, err := verifier.DetectHardwareBinding(context.Background())
	if err != nil {
		log.Fatalf("resolve expected device binding: %v", err)
	}
	fmt.Printf("device binding: platform=%s method=%s pub_hash=%s\n", expectedDevice.Platform, expectedDevice.Method, expectedDevice.PubHash)

	now := time.Now().UTC()
	timeGuardKey := mustLoadEmbeddedClockGuardKey()
	timeGuard := licensepkg.NewFileTimeGuardWithMAC(clockStatePath, 2*time.Minute, timeGuardKey)
	if err := timeGuard.CheckAndUpdate(context.Background(), now); err != nil {
		log.Fatalf("clock rollback check failed: %v", err)
	}

	payload, err := verifier.Verify(context.Background(), token, expectedDevice, now)
	if err != nil {
		log.Fatalf("verify license: %v", err)
	}

	fmt.Printf("license verified: id=%s project=%s customer=%s claims=%v\n",
		payload.LicenseID,
		payload.ProjectID,
		payload.CustomerID,
		payload.X,
	)
}

func loadPublicKeys(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var keys map[string]string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func mustLoadEmbeddedClockGuardKey() []byte {
	key, err := base64.RawURLEncoding.DecodeString(embeddedClockGuardKeyB64)
	if err != nil {
		log.Fatalf("invalid embedded clock guard key: %v", err)
	}
	if len(key) < 16 {
		log.Fatalf("clock guard key too short: need at least 16 bytes")
	}
	return key
}

func chooseLicensePath(base string) string {
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}

	candidates := []string{
		filepath.Join(base, "license.json"),
		filepath.Join(base, fmt.Sprintf("license.%s.json", platform)),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}
