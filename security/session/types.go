package session

import "time"

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type Claims struct {
	Subject   string
	TokenType string
	JTI       string
	ExpiresAt time.Time
	Values    map[string]any
}

func (c Claims) Value(key string) (any, bool) {
	v, ok := c.Values[key]
	return v, ok
}
