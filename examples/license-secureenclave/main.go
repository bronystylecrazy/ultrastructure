//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	licensepkg "github.com/bronystylecrazy/ultrastructure/security/license"
)

func main() {
	ctx := context.Background()

	signer := licensepkg.NewMacOSKeychainSecureEnclaveSigner("", true)
	binding, err := licensepkg.ExpectedHardwareBindingFromSecureEnclave(ctx, signer)
	if err != nil {
		fmt.Printf("secure-enclave binding unavailable: %v\n", err)
		fmt.Println("Tip: OSStatus -34018 usually means this process lacks a usable keychain/entitlement context.")
		fmt.Println("Run from a signed app context or grant keychain access, then retry.")
		return
	}

	bindingJSON, _ := json.MarshalIndent(binding, "", "  ")
	fmt.Printf("runtime secure-enclave binding (use this during license issuance):\n%s\n", string(bindingJSON))

	base := "examples/license-secureenclave/testdata"
	publicKeysPath := filepath.Join(base, "public_keys.json")
	licensePath := filepath.Join(base, "license.json")

	if !fileExists(publicKeysPath) || !fileExists(licensePath) {
		fmt.Printf(
			"\nNo local test license found.\nCreate %s and %s, then rerun to verify.\n",
			publicKeysPath,
			licensePath,
		)
		return
	}

	publicKeys, err := loadPublicKeys(publicKeysPath)
	if err != nil {
		log.Fatalf("load public keys: %v", err)
	}
	verifier, err := licensepkg.NewVerifier(
		licensepkg.WithPublicKeyProvider(licensepkg.StaticPublicKeyProvider(publicKeys)),
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

	payload, err := verifier.VerifyWithSecureEnclaveChallenge(ctx, signer, token, time.Now().UTC())
	if err != nil {
		log.Fatalf("verify license + challenge: %v", err)
	}

	fmt.Printf("license verified: id=%s project=%s customer=%s claims=%v\n",
		payload.LicenseID,
		payload.ProjectID,
		payload.CustomerID,
		payload.X,
	)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
