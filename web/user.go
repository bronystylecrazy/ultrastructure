package web

import (
	"context"
	"time"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type User struct {
	Sub string    `json:"sub"`
	Sid string    `json:"sid,omitempty"`
	Jti string    `json:"jti,omitempty"`
	Exp time.Time `json:"exp"`
}

// WithUser stores User in the context
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// GetUserFromContext extracts User from context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(UserContextKey).(*User)
	return user, ok
}
