package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	licensepkg "github.com/bronystylecrazy/ultrastructure/security/license"
)

const demoSeedHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func main() {
	var (
		outPath       string
		projectID     string
		customerID    string
		licenseID     string
		kid           string
		expiresInDays int
		neverExpires  bool
		seedHex       string
	)

	flag.StringVar(&outPath, "out", "examples/license-file/testdata/license.json", "output license JSON path")
	flag.StringVar(&projectID, "project", "ultrastructure-demo", "project id")
	flag.StringVar(&customerID, "customer", "customer-test", "customer id")
	flag.StringVar(&licenseID, "license-id", "", "license id (auto-generated if empty)")
	flag.StringVar(&kid, "kid", "kid-test-2026-02", "key id")
	flag.IntVar(&expiresInDays, "expires-days", 365, "expiry days from now")
	flag.BoolVar(&neverExpires, "never-expires", false, "generate non-expiring license")
	flag.StringVar(&seedHex, "seed-hex", demoSeedHex, "hex-encoded ed25519 seed (32 bytes) for signing")
	flag.Parse()

	if licenseID == "" {
		licenseID = "lic-" + uuid.NewString()
	}

	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		log.Fatalf("invalid seed-hex: %v", err)
	}
	if len(seed) != ed25519.SeedSize {
		log.Fatalf("invalid seed length: got %d, want %d", len(seed), ed25519.SeedSize)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)

	ctx := context.Background()
	deviceBinding, err := licensepkg.ExpectedDeviceBinding(ctx)
	if err != nil {
		log.Fatalf("resolve runtime device binding: %v", err)
	}

	now := time.Now().UTC()
	var expiry *int64
	if !neverExpires {
		value := now.Add(time.Duration(expiresInDays) * 24 * time.Hour).Unix()
		expiry = &value
	}

	payload := licensepkg.LicensePayload{
		V:            1,
		LicenseID:    licenseID,
		ProjectID:    projectID,
		CustomerID:   customerID,
		IssuedAt:     now.Unix(),
		Expiry:       expiry,
		NeverExpires: neverExpires,
		KID:          kid,
		DeviceBind:   *deviceBinding,
		X: map[string]any{
			"max_cameras": 4,
			"plan":        "pro",
		},
		Nonce: "nonce-" + uuid.NewString(),
	}

	unsigned := payload
	unsigned.Sig = ""
	rawUnsigned, err := json.Marshal(unsigned)
	if err != nil {
		log.Fatalf("marshal unsigned payload: %v", err)
	}
	payload.Sig = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, rawUnsigned))

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("marshal signed payload: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		log.Fatalf("write license file: %v", err)
	}

	fmt.Printf("license written: %s\n", outPath)
	fmt.Printf("platform=%s method=%s pub_hash=%s\n", payload.DeviceBind.Platform, payload.DeviceBind.Method, payload.DeviceBind.PubHash)
	fmt.Printf("kid=%s license_id=%s never_expires=%v\n", payload.KID, payload.LicenseID, payload.NeverExpires)
}
