package main

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	apikey "github.com/bronystylecrazy/ultrastructure/security/apikey"
	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	authz "github.com/bronystylecrazy/ultrastructure/security/authz"
	"github.com/bronystylecrazy/ultrastructure/security/session"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/bronystylecrazy/ultrastructure/x/autoswag"
	"github.com/gofiber/fiber/v3"
)

type memoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*apikey.StoredKey
}

func NewMemoryKeyStore() *memoryKeyStore {
	return &memoryKeyStore{
		keys: map[string]*apikey.StoredKey{},
	}
}

func (s *memoryKeyStore) FindByKeyID(ctx context.Context, keyID string) (*apikey.StoredKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.keys[keyID]
	if !ok {
		return nil, nil
	}
	return cloneStoredKey(v), nil
}

func (s *memoryKeyStore) MarkUsed(ctx context.Context, keyID string, usedAt time.Time) error {
	return nil
}

func (s *memoryKeyStore) RevokeKey(ctx context.Context, keyID string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.keys[keyID]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	v.RevokedAt = &now
	return nil
}

func (s *memoryKeyStore) SaveIssued(issued *apikey.IssuedKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[issued.KeyID] = &apikey.StoredKey{
		KeyID:      issued.KeyID,
		AppID:      issued.AppID,
		SecretHash: issued.SecretHash,
		Scopes:     append([]string(nil), issued.Scopes...),
		Metadata:   cloneMap(issued.Metadata),
		ExpiresAt:  issued.ExpiresAt,
	}
}

type handler struct {
	userIssuer session.Issuer
	userAuth   session.Validator
	appKey     apikey.Manager
	store      *memoryKeyStore
}

func NewHandler(userIssuer session.Issuer, userAuth session.Validator, appKey apikey.Manager, store *memoryKeyStore) *handler {
	return &handler{
		userIssuer: userIssuer,
		userAuth:   userAuth,
		appKey:     appKey,
		store:      store,
	}
}

func (h *handler) Handle(r web.Router) {
	r.Post("/api/v1/auth/login", h.login)
	r.Post("/api/v1/apikeys", h.issueAPIKey)
	r.Delete("/api/v1/apikeys/:key_id", h.revokeAPIKey)

	p := r.Group(
		"/api/v1/integration",
		authn.Any(
			authn.UserTokenAuthenticator(h.userAuth),
			authn.APIKeyAuthenticator(h.appKey),
		),
		authz.ResolvePolicy(authz.PolicyPreferUser),
	)
	p.Get("/resource", h.readResource).
		With(
			authz.Policy("resource.read"),
			authz.Scope("resource:read"),
		)
	p.Post("/resource", h.writeResource).
		With(authz.Policy("resource.write"))
}

func (h *handler) login(c fiber.Ctx) error {
	// Demo-only: issue token for a fixed user.
	pair, err := h.userIssuer.Generate("user-1", session.WithAccessClaims(map[string]any{
		"role": "admin",
	}))
	if err != nil {
		return err
	}
	return c.JSON(pair)
}

func (h *handler) issueAPIKey(c fiber.Ctx) error {
	// Demo-only: issue one API key with read scope.
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	issued, err := h.appKey.IssueKey(
		"partner-app-1",
		apikey.WithPrefix("intg"),
		apikey.WithScopes("read:resource"),
		apikey.WithMetadata(map[string]string{"env": "dev"}),
		apikey.WithExpiresAt(&expiresAt),
	)
	if err != nil {
		return err
	}

	// Persist hash + metadata (never persist raw key).
	h.store.SaveIssued(issued)

	return c.JSON(fiber.Map{
		"key_id":     issued.KeyID,
		"app_id":     issued.AppID,
		"raw_key":    issued.RawKey, // show once
		"scopes":     issued.Scopes,
		"expires_at": issued.ExpiresAt,
	})
}

func (h *handler) revokeAPIKey(c fiber.Ctx) error {
	keyID := c.Params("key_id")
	if err := h.appKey.RevokeKey(c.Context(), keyID, "manual-revoke"); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handler) readResource(c fiber.Ctx) error {
	p, ok := authn.PrincipalFromContext(c.Context())
	if !ok || p == nil {
		return sendUnauthorized(c, "unauthorized")
	}

	switch p.Type {
	case authn.PrincipalUser:
		// user authz path
		if !slices.Contains(p.Roles, "admin") {
			return sendForbidden(c, "forbidden")
		}
	case authn.PrincipalApp:
		// app authz path
		if !slices.Contains(p.Scopes, "read:resource") {
			return sendForbidden(c, "forbidden")
		}
	default:
		return sendForbidden(c, "forbidden")
	}

	return c.JSON(fiber.Map{
		"principal_type": p.Type,
		"subject":        p.Subject,
		"app_id":         p.AppID,
		"resource":       "read-ok",
	})
}

func (h *handler) writeResource(c fiber.Ctx) error {
	p, ok := authn.PrincipalFromContext(c.Context())
	if !ok || p == nil {
		return sendUnauthorized(c, "unauthorized")
	}

	switch p.Type {
	case authn.PrincipalUser:
		if !slices.Contains(p.Roles, "admin") {
			return sendForbidden(c, "forbidden")
		}
	case authn.PrincipalApp:
		if !slices.Contains(p.Scopes, "write:resource") {
			return sendForbidden(c, "forbidden")
		}
	default:
		return sendForbidden(c, "forbidden")
	}

	return c.JSON(fiber.Map{
		"principal_type": p.Type,
		"result":         "write-ok",
	})
}

func cloneStoredKey(in *apikey.StoredKey) *apikey.StoredKey {
	if in == nil {
		return nil
	}
	out := *in
	if in.Scopes != nil {
		out.Scopes = append([]string(nil), in.Scopes...)
	}
	if in.Metadata != nil {
		out.Metadata = cloneMap(in.Metadata)
	}
	if in.ExpiresAt != nil {
		t := *in.ExpiresAt
		out.ExpiresAt = &t
	}
	if in.RevokedAt != nil {
		t := *in.RevokedAt
		out.RevokedAt = &t
	}
	return &out
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sendUnauthorized(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(web.Error{
		Error: web.ErrorDetail{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	})
}

func sendForbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(web.Error{
		Error: web.ErrorDetail{
			Code:    "FORBIDDEN",
			Message: message,
		},
	})
}

func main() {
	store := NewMemoryKeyStore()

	if err := us.New(
		apikey.Providers(
			apikey.UseLookup(store),
			apikey.UseUsageRecorder(store),
			apikey.UseRevoker(store),
		),
		cmd.UseBasicCommands(),
		cmd.Run(
			web.Init(),
			autoswag.Use(
				autoswag.WithBearerSecurityScheme("BearerAuth"),
				autoswag.WithAPIKeySecurityScheme("ApiKeyAuth", "X-API-Key", "header"),
			),
			authz.UseScopeGovernance(
				authz.ScopeDefinition{Name: "read:resource", Description: "Read integration resources"},
				authz.ScopeDefinition{Name: "write:resource", Description: "Write integration resources"},
			),
			authz.UsePolicyGovernance(
				authz.PolicyDefinition{
					Name:        "resource.read",
					Description: "Read integration resources",
					AllScopes:   []string{"read:resource"},
				},
				authz.PolicyDefinition{
					Name:        "resource.write",
					Description: "Write integration resources",
					AllScopes:   []string{"write:resource"},
				},
			),
			session.UseRefreshRoute(
				"/api/v1/auth/refresh",
				session.WithRefreshSubjectResolver(session.SubjectFromContext),
			),
			authz.UseScopeCatalogRoute("/api/v1/authz/scopes"),
			di.Supply(store),
			di.Provide(NewHandler),
			di.Invoke(func() {
				fmt.Println("authn-mixed example running")
				fmt.Println("1) POST /api/v1/auth/login")
				fmt.Println("2) POST /api/v1/apikeys")
				fmt.Println("3) GET  /api/v1/integration/resource")
				fmt.Println("   - Bearer <access_token> OR ApiKey <raw_key>")
				fmt.Println("4) Swagger UI: GET /docs")
				fmt.Println("   OpenAPI JSON: GET /docs/swagger.json")
			}),
		),
	).Run(); err != nil {
		panic(err)
	}
}
