package license

import (
	"encoding/json"
	"time"
)

// ─── Input DTOs ────────────────────────────────────────────────────────────────
type DeviceBinding struct {
	Platform string `json:"platform"` // windows|linux|macos
	Method   string `json:"method"`   // tpm-ek-hash|tpm-ak-hash|secure-enclave-key|os-keystore
	PubHash  string `json:"pub_hash"` // b64url(sha256(pub))
}

type MintTokenOneInput struct {
	ProjectID  string
	CustomerID string
	Prefix     string
	ExpiresAt  *time.Time
	IdemKey    string // optional (recommended)
}

type ActivateInput struct {
	ProjectID     string
	CustomerID    string
	Token         string
	DeviceBinding DeviceBinding
	NeverExpires  bool
	ExpiresAt     *time.Time
	X             json.RawMessage // claims
	IdemKey       string          // optional (recommended)
}

type IssueDirectInput struct {
	ProjectID     string
	CustomerID    string
	DeviceBinding DeviceBinding
	NeverExpires  bool
	ExpiresAt     *time.Time
	KID           string          // optional; fallback to current signing KID
	X             json.RawMessage // claims
	IdemKey       string          // optional
}

// ─── Output DTOs ───────────────────────────────────────────────────────────────
type TokenResult struct {
	Token      string
	ProjectID  string
	CustomerID string
	ExpiresAt  *time.Time
}

type LicensePayload struct {
	V            int            `json:"v"`
	LicenseID    string         `json:"license_id"`
	ProjectID    string         `json:"project_id"`
	CustomerID   string         `json:"customer_id"`
	IssuedAt     int64          `json:"issued_at"`
	Expiry       *int64         `json:"expiry,omitempty"`
	NeverExpires bool           `json:"never_expires,omitempty"`
	KID          string         `json:"kid"`
	DeviceBind   DeviceBinding  `json:"device_binding"`
	X            map[string]any `json:"x,omitempty"`
	Nonce        string         `json:"nonce"`
	Sig          string         `json:"sig,omitempty"`
}
