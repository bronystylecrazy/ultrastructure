package jws

import (
	"fmt"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

type Verifier interface {
	Verify(tokenValue string) (Claims, error)
}

var _ Verifier = (*JWTSigner)(nil)

func (s *JWTSigner) Verify(tokenValue string) (Claims, error) {
	token, err := jwtgo.Parse(tokenValue, func(token *jwtgo.Token) (any, error) {
		gotAlg := ""
		if token != nil && token.Method != nil {
			gotAlg = token.Method.Alg()
		}
		if gotAlg != s.signingAlg {
			return nil, fmt.Errorf("%w: got=%s want=%s", ErrUnexpectedTokenAlg, gotAlg, s.signingAlg)
		}
		return s.verifyKey, nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !token.Valid {
		return Claims{}, jwtgo.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwtgo.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidClaims
	}
	return claimsFromJWT(claims), nil
}
