package session

import (
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/security/jws"
	"github.com/bronystylecrazy/ultrastructure/x/paseto"
)

type JWTManagerOption func(*JWTManager)

const JWTManagerOptionsGroupName = "us/session/jwt_manager_options"

func UseJWTManagerOptions(opts ...JWTManagerOption) di.Node {
	nodes := make([]any, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		option := opt
		nodes = append(nodes, di.Provide(func() JWTManagerOption {
			return option
		}, di.Group(JWTManagerOptionsGroupName)))
	}
	return di.Options(nodes...)
}

func WithAccessExtractors(extractors ...Extractor) JWTManagerOption {
	return func(m *JWTManager) {
		m.SetDefaultAccessExtractors(extractors...)
	}
}

func WithRefreshExtractors(extractors ...Extractor) JWTManagerOption {
	return func(m *JWTManager) {
		m.SetDefaultRefreshExtractors(extractors...)
	}
}

type RefreshRouteOption func(*RefreshHandler)

func WithRefreshPairDeliverer(deliverer PairDeliverer) RefreshRouteOption {
	return func(h *RefreshHandler) {
		h.WithDeliverer(deliverer)
	}
}

func WithRefreshPairDelivererResolver(resolver PairDelivererResolver) RefreshRouteOption {
	return func(h *RefreshHandler) {
		h.WithDelivererResolver(resolver)
	}
}

func WithRefreshSubjectResolver(subjectResolver SubjectResolver) RefreshRouteOption {
	return func(h *RefreshHandler) {
		h.WithSubjectResolver(subjectResolver)
	}
}

func UseRefreshRoute(path string, opts ...RefreshRouteOption) di.Node {
	return di.Provide(func(service Manager) *RefreshHandler {
		h := NewRefreshHandler(service, nil).
			WithPath(path).
			WithDelivererResolver(WebCookieOrJSONPairDelivererResolver())
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			opt(h)
		}
		return h
	})
}

func UseRevocationStore(store RevocationStore) di.Node {
	return di.Supply(store, di.AsSelf[RevocationStore]())
}

// UseSignerVerifier supplies a custom SignerVerifier for session management.
// This allows using PASETO or other token formats instead of JWT.
func UseSignerVerifier(signerVerifier jws.SignerVerifier) di.Node {
	return di.Supply(signerVerifier, di.As[jws.Signer](), di.As[jws.Verifier](), di.As[jws.SignerVerifier]())
}

// UsePasetoConfig configures session to use PASETO with the given config.
// This supplies the jws.Config type that session.Provider expects.
func UsePasetoConfig(secret string, opts ...PasetoConfigOption) di.Node {
	cfg := pasetoConfigToJWSConfig(secret, opts...)
	return di.Supply(cfg, di.AsSelf[jws.Config]())
}

// PasetoConfigOption configures PASETO-based session.
type PasetoConfigOption func(*pasetoConfig)

type pasetoConfig struct {
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

func pasetoConfigToJWSConfig(secret string, opts ...PasetoConfigOption) jws.Config {
	cfg := &pasetoConfig{
		accessTokenTTL:  defaultAccessTokenTTL,
		refreshTokenTTL: defaultRefreshTokenTTL,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return jws.Config{
		Secret:          secret,
		AccessTokenTTL:  cfg.accessTokenTTL,
		RefreshTokenTTL: cfg.refreshTokenTTL,
		Issuer:          cfg.issuer,
		Algorithm:       "HS256",
	}
}

// WithPasetoAccessTokenTTL sets the access token TTL.
func WithPasetoAccessTokenTTL(d time.Duration) PasetoConfigOption {
	return func(c *pasetoConfig) {
		c.accessTokenTTL = d
	}
}

// WithPasetoRefreshTokenTTL sets the refresh token TTL.
func WithPasetoRefreshTokenTTL(d time.Duration) PasetoConfigOption {
	return func(c *pasetoConfig) {
		c.refreshTokenTTL = d
	}
}

// WithPasetoIssuer sets the issuer claim.
func WithPasetoIssuer(issuer string) PasetoConfigOption {
	return func(c *pasetoConfig) {
		c.issuer = issuer
	}
}

// UsePaseto configures the session to use PASETO tokens.
// It provides a PasetoManager that implements the Manager interface.
func UsePaseto(config paseto.Config, opts ...PasetoManagerOption) di.Node {
	return di.Options(
		di.Supply(config, di.AsSelf[paseto.Config]()),
		di.Provide(paseto.New, di.AsSelf[paseto.Signer](), di.AsSelf[paseto.Verifier](), di.AsSelf[paseto.SignerVerifier]()),
		pasetoManagerProvider(opts...),
	)
}

// UsePasetoWithSigner configures the session to use PASETO with a pre-configured signer.
func UsePasetoWithSigner(config paseto.Config, signer paseto.SignerVerifier, opts ...PasetoManagerOption) di.Node {
	return di.Options(
		di.Supply(signer, di.AsSelf[paseto.Signer](), di.AsSelf[paseto.Verifier](), di.AsSelf[paseto.SignerVerifier]()),
		pasetoManagerProvider(opts...),
	)
}

// PasetoManagerOption configures a PasetoManager.
type PasetoManagerOption func(*PasetoManager)

// WithPasetoAccessExtractors sets access token extractors.
func WithPasetoAccessExtractors(extractors ...Extractor) PasetoManagerOption {
	return func(m *PasetoManager) {
		m.SetDefaultAccessExtractors(extractors...)
	}
}

// WithPasetoRefreshExtractors sets refresh token extractors.
func WithPasetoRefreshExtractors(extractors ...Extractor) PasetoManagerOption {
	return func(m *PasetoManager) {
		m.SetDefaultRefreshExtractors(extractors...)
	}
}

func pasetoManagerProvider(opts ...PasetoManagerOption) di.Node {
	return di.Provide(func(cfg paseto.Config, signer paseto.SignerVerifier, cacheStore RevocationCache, customStore RevocationStore) (*PasetoManager, error) {
		manager, err := NewPasetoManager(cfg, signer)
		if err != nil {
			return nil, err
		}
		for _, opt := range opts {
			if opt != nil {
				opt(manager)
			}
		}
		if customStore != nil {
			manager.SetRevocationStore(customStore)
		} else if cacheStore != nil {
			namespace := ""
			if cfg.Issuer != "" {
				namespace = cfg.Issuer
			}
			manager.SetRevocationStore(NewRevocationStoreWithNamespace(cacheStore, "", namespace))
		}
		return manager, nil
	}, di.AsSelf[Manager](), di.AsSelf[Issuer](), di.AsSelf[Validator](), di.AsSelf[Revoker](), di.AsSelf[Rotator](), di.AsSelf[MiddlewareFactory]())
}
