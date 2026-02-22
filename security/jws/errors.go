package jws

import "errors"

var ErrMissingSecret = errors.New("token: missing secret")
var ErrMissingPrivateKey = errors.New("token: missing private key")
var ErrMissingPublicKey = errors.New("token: missing public key")
var ErrInvalidPrivateKey = errors.New("token: invalid private key")
var ErrInvalidPublicKey = errors.New("token: invalid public key")
var ErrReadPrivateKeyFile = errors.New("token: read private key file")
var ErrReadPublicKeyFile = errors.New("token: read public key file")
var ErrUnsupportedAlg = errors.New("token: unsupported signing algorithm")
var ErrInvalidClaims = errors.New("token: invalid claims")
var ErrUnexpectedTokenAlg = errors.New("token: unexpected signing method")
