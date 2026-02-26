package license

type HardwareBinding struct {
	Platform string `json:"platform"` // windows|linux|macos
	Method   string `json:"method"`   // tpm-ek-hash|tpm-ak-hash|secure-enclave-key|os-keystore
	PubHash  string `json:"pub_hash"` // b64url(sha256(pub))
}

type LicensePayload struct {
	V            int             `json:"v"`
	LicenseID    string          `json:"license_id"`
	ProjectID    string          `json:"project_id"`
	CustomerID   string          `json:"customer_id"`
	IssuedAt     int64           `json:"issued_at"`
	Expiry       *int64          `json:"expiry,omitempty"`
	NeverExpires bool            `json:"never_expires,omitempty"`
	KID          string          `json:"kid"`
	HardwareBind HardwareBinding `json:"hardware_binding"`
	X            map[string]any  `json:"x,omitempty"`
	Nonce        string          `json:"nonce"`
	Sig          string          `json:"sig,omitempty"`
}
