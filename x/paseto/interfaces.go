package paseto

// Signer signs PASETO tokens.
type Signer interface {
	Sign(claims map[string]any) (string, error)
}

// Verifier verifies PASETO tokens.
type Verifier interface {
	Verify(tokenValue string) (Claims, error)
}

// SignerVerifier combines Signer and Verifier interfaces.
type SignerVerifier interface {
	Signer
	Verifier
}
