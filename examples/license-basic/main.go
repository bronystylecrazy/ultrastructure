package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	licensepkg "github.com/bronystylecrazy/ultrastructure/security/license"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("generate key: %v", err)
	}

	const kid = "demo-kid"
	keys := licensepkg.StaticPublicKeyProvider{
		kid: base64.RawURLEncoding.EncodeToString(pub),
	}
	verifier, err := licensepkg.NewVerifier(
		licensepkg.WithPublicKeyProvider(keys),
		licensepkg.WithHardwareDetector(licensepkg.NewHardwareDetector()),
	)
	if err != nil {
		log.Fatalf("new verifier: %v", err)
	}

	now := time.Now().UTC()
	expiry := now.Add(24 * time.Hour).Unix()
	device := licensepkg.HardwareBinding{
		Platform: "linux",
		Method:   "tpm-ek-hash",
		PubHash:  "demo-device-pub-hash",
	}

	payload := licensepkg.LicensePayload{
		V:            1,
		LicenseID:    "lic-demo-001",
		ProjectID:    "project-ultrastructure",
		CustomerID:   "customer-acme",
		IssuedAt:     now.Unix(),
		Expiry:       &expiry,
		KID:          kid,
		HardwareBind: device,
		X: map[string]any{
			"max_cameras": 4,
			"plan":        "pro",
		},
		Nonce: "nonce-demo-001",
	}

	token, err := signToken(payload, priv)
	if err != nil {
		log.Fatalf("sign token: %v", err)
	}

	verified, err := verifier.Verify(context.Background(), token, &device, now)
	if err != nil {
		log.Fatalf("verify failed: %v", err)
	}

	fmt.Printf("license verified: license_id=%s project_id=%s customer_id=%s claims=%v\n",
		verified.LicenseID,
		verified.ProjectID,
		verified.CustomerID,
		verified.X,
	)
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
