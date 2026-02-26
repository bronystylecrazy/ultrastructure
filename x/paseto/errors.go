package paseto

import "errors"

var ErrMissingSecret = errors.New("paseto: missing secret")
var ErrInvalidSecret = errors.New("paseto: invalid secret length")
var ErrInvalidToken = errors.New("paseto: invalid token")
var ErrExpiredToken = errors.New("paseto: token expired")
var ErrInvalidClaims = errors.New("paseto: invalid claims")
var ErrUnexpectedTokenVersion = errors.New("paseto: unexpected token version")
var ErrMissingFooter = errors.New("paseto: missing required footer")
var ErrInvalidFooter = errors.New("paseto: invalid footer")
