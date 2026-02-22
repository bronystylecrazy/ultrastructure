package session

import "errors"

var ErrRevocationStoreNotConfigured = errors.New("token: revocation store not configured")
var ErrSignerNotConfigured = errors.New("token: signer not configured")
var ErrRevocationCacheMiss = errors.New("token: revocation cache miss")
var ErrMissingTokenJTI = errors.New("token: missing jti in token")
var ErrMissingTokenExp = errors.New("token: missing exp in token")
var ErrTokenRevoked = errors.New("token: token revoked")
var ErrMissingRefreshSubjectResolver = errors.New("token: missing refresh subject resolver")
var ErrInvalidClaims = errors.New("token: invalid claims")
var ErrInvalidTokenType = errors.New("token: invalid token type")
var ErrMissingTokenSub = errors.New("token: missing subject in token")
var ErrTokenMissingInContext = errors.New("token: token missing from fiber context")
