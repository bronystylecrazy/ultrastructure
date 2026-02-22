package jws

type SignerVerifier interface {
	Signer
	Verifier
}

var _ SignerVerifier = (*JWTSigner)(nil)
