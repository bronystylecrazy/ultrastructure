package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	licensepkg "github.com/bronystylecrazy/ultrastructure/security/license"
)

const demoSeedHex = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: %s <issue|verify> [flags]", os.Args[0])
	}

	switch os.Args[1] {
	case "issue":
		runIssue(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	default:
		fatalf("unknown command %q (expected issue or verify)", os.Args[1])
	}
}

func runIssue(args []string) {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	outPath := fs.String("out", "./license.json", "output license json path")
	projectID := fs.String("project", "ultrastructure-demo", "project id")
	customerID := fs.String("customer", "customer-demo", "customer id")
	licenseID := fs.String("license-id", "lic-demo-001", "license id")
	kid := fs.String("kid", "kid-demo-001", "key id")
	expiresDays := fs.Int("expires-days", 365, "expiry days from now")
	neverExpires := fs.Bool("never-expires", false, "non-expiring license")
	seedHex := fs.String("seed-hex", demoSeedHex, "32-byte ed25519 seed hex")
	if err := fs.Parse(args); err != nil {
		fatalf("parse flags: %v", err)
	}

	seed, err := hex.DecodeString(strings.TrimSpace(*seedHex))
	if err != nil {
		fatalf("invalid seed-hex: %v", err)
	}
	if len(seed) != ed25519.SeedSize {
		fatalf("invalid seed length: got %d want %d", len(seed), ed25519.SeedSize)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	binding, err := licensepkg.NewHardwareDetector().Detect(context.Background())
	if err != nil {
		fatalf("detect hardware binding: %v", err)
	}

	now := time.Now().UTC()
	var expiry *int64
	if !*neverExpires {
		v := now.Add(time.Duration(*expiresDays) * 24 * time.Hour).Unix()
		expiry = &v
	}

	payload := licensepkg.LicensePayload{
		V:            1,
		LicenseID:    *licenseID,
		ProjectID:    *projectID,
		CustomerID:   *customerID,
		IssuedAt:     now.Unix(),
		Expiry:       expiry,
		NeverExpires: *neverExpires,
		KID:          *kid,
		HardwareBind: *binding,
		X: map[string]any{
			"plan": "pro",
		},
		Nonce: fmt.Sprintf("nonce-%d", now.UnixNano()),
	}

	token, err := signToken(payload, priv)
	if err != nil {
		fatalf("sign token: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	if err := os.WriteFile(*outPath, []byte(token), 0o644); err != nil {
		fatalf("write license file: %v", err)
	}

	publicKeys := map[string]string{*kid: base64.RawURLEncoding.EncodeToString(pub)}
	publicKeysJSON, err := json.Marshal(publicKeys)
	if err != nil {
		fatalf("marshal public keys json: %v", err)
	}
	publicKeysB64 := base64.RawURLEncoding.EncodeToString(publicKeysJSON)

	fmt.Printf("license written: %s\n", *outPath)
	fmt.Printf("device_binding: platform=%s method=%s pub_hash=%s\n", binding.Platform, binding.Method, binding.PubHash)
	fmt.Printf("PUBLIC_KEYS_JSON=%s\n", string(publicKeysJSON))
	fmt.Printf("PUBLIC_KEYS_B64=%s\n", publicKeysB64)
	fmt.Printf("build verifier with:\n")
	fmt.Printf("go build -ldflags \"-X 'github.com/bronystylecrazy/ultrastructure/security/license.PublicKeysB64=%s'\" -o ./license-e2e ./examples/license-e2e\n", publicKeysB64)
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	licensePath := fs.String("license", "./license.json", "license file path")
	if err := fs.Parse(args); err != nil {
		fatalf("parse flags: %v", err)
	}

	tokenBytes, err := os.ReadFile(*licensePath)
	if err != nil {
		fatalf("read license file: %v", err)
	}

	payload, err := licensepkg.VerifyOfflineSingleDevice(context.Background(), string(tokenBytes), time.Now().UTC())
	if err != nil {
		if errors.Is(err, licensepkg.ErrNoPublicKeysConfigured) {
			fatalf("verify failed: no embedded public keys; rebuild with -ldflags -X security/license.PublicKeysB64")
		}
		fatalf("verify failed: %v", err)
	}

	fmt.Printf("verified: license_id=%s project_id=%s customer_id=%s claims=%v\n", payload.LicenseID, payload.ProjectID, payload.CustomerID, payload.X)
}

func signToken(payload licensepkg.LicensePayload, priv ed25519.PrivateKey) (string, error) {
	unsigned := payload
	unsigned.Sig = ""

	rawUnsigned, err := json.Marshal(unsigned)
	if err != nil {
		return "", err
	}

	payload.Sig = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, rawUnsigned))
	rawSigned, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(rawSigned), nil
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
