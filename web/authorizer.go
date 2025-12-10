package web

import (
	"fmt"
	"time"

	"github.com/bronystylecrazy/flexinfra/config"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	AuthorizerContextKey   = "user"
	accessTokenCookieKey   = "access_token"
	refreshTokenCookieKey  = "refresh_token"
	tokenLookupHeaderValue = "header:Authorization"
)

type authorizer struct {
	jwtConfig      config.JwtConfig
	log            *zap.Logger
	tokenManager   TokenManager
	sessionManager SessionManager
	tokenCache     caching.TokenCache
}

func NewAuthorizer(
	jwtConfig config.JwtConfig,
	log *zap.Logger,
	tokenManager TokenManager,
	sessionManager SessionManager,
	tokenCache caching.TokenCache,
) Authorizer {
	return &authorizer{
		jwtConfig:      jwtConfig,
		log:            log,
		tokenManager:   tokenManager,
		sessionManager: sessionManager,
		tokenCache:     tokenCache,
	}
}

func (a *authorizer) Authorize() fiber.Handler {
	cookieConf := jwtware.Config{
		SigningKey:  jwtware.SigningKey{Key: []byte(a.jwtConfig.Secret)},
		TokenLookup: fmt.Sprintf("cookie:%s,%s", accessTokenCookieKey, tokenLookupHeaderValue),
		ContextKey:  AuthorizerContextKey,
		SuccessHandler: func(c *fiber.Ctx) error {
			// Get JWT token from context
			tokenValue := c.Locals(AuthorizerContextKey)
			token, ok := tokenValue.(*jwt.Token)
			if !ok || token == nil {
				a.log.Warn("JWT token missing or invalid type",
					zap.Any("token_value", tokenValue),
				)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Unauthorized",
				})
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				a.log.Warn("Invalid JWT claims format")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Unauthorized",
				})
			}

			// Extract User from JWT claims
			user, err := a.extractUserFromClaims(claims)
			if err != nil {
				a.log.Warn("Failed to extract user from claims", zap.Error(err))
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token claims",
				})
			}

			// Check if token is blacklisted (JTI check)
			if user.Jti != "" {
				blacklisted, err := a.tokenCache.IsTokenBlacklisted(c.UserContext(), user.Jti)
				if err != nil {
					a.log.Warn("Failed to check token blacklist", zap.String("jti", user.Jti), zap.Error(err))
				} else if blacklisted {
					a.log.Warn("Blacklisted token attempted access", zap.String("jti", user.Jti))
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Token has been revoked",
					})
				}
			}

			// Check if token has session_id claim
			if user.Sid != "" {
				// Validate session (with built-in caching)
				err := a.sessionManager.ValidateUserSession(c.UserContext(), user.Sid)
				if err != nil {
					a.log.Warn("Session validation failed",
						zap.String("session_id", user.Sid),
						zap.Error(err))
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Session expired or invalid",
					})
				}
				a.log.Debug("Session validated successfully", zap.String("session_id", user.Sid))
			}

			ctx := WithUser(c.Context(), user)
			c.SetUserContext(ctx)

			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			a.log.Warn("Unauthorized access attempt", zap.Error(err))
			refreshToken := c.Cookies(refreshTokenCookieKey)
			if refreshToken != "" {
				// Attempt to renew the access token using the refresh token
				result, err := a.tokenManager.RenewAccessToken(c.UserContext(), refreshToken)
				if err != nil {
					a.log.Error("Failed to renew access token", zap.Error(err))
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Unauthorized",
					})
				}

				token, err := jwt.Parse(result.AccessToken, func(token *jwt.Token) (any, error) {
					if token.Method != jwt.SigningMethodHS256 {
						return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					}
					return []byte(a.jwtConfig.Secret), nil
				})
				if err != nil {
					a.log.Error("Failed to parse renewed access token", zap.Error(err))
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Unauthorized",
					})
				}

				c.Locals(AuthorizerContextKey, token)

				// Set the new access token in the response cookies
				c.Cookie(&fiber.Cookie{
					Name:     accessTokenCookieKey,
					Value:    result.AccessToken,
					Expires:  time.Now().Add(AccessTokenExpiry),
					HTTPOnly: true,
					Secure:   true,
					SameSite: "Lax",
				})

				c.Set("X-Token-Renewed", "true") // Optional: Indicate that the token was renewed

				return c.Next()
			}

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		},
	}
	return jwtware.New(cookieConf)
}

// extractUserFromClaims extracts User from JWT claims
func (a *authorizer) extractUserFromClaims(claims jwt.MapClaims) (*User, error) {
	// Extract required fields
	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user ID in token")
	}

	sessionID, _ := claims["session_id"].(string) // Optional
	jti, _ := claims["jti"].(string)              // Optional

	// Extract expiration time
	var expiresAt time.Time
	if exp, ok := claims["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0)
	}

	return &User{
		Sub: userID,
		Sid: sessionID,
		Jti: jti,
		Exp: expiresAt,
	}, nil
}
