package session

import (
	"strings"

	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/security/jws"
	"go.uber.org/fx"
)

func Providers(opts ...di.Node) di.Node {
	nodes := []any{
		cfg.Config[jws.Config]("jwt", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		di.Provide(jws.NewSigner, di.As[jws.Signer](), di.As[jws.Verifier](), di.As[jws.SignerVerifier]()),
		di.Provide(
			newJWTManagerWithDefaultRevocation,
			di.AsSelf[Manager](),
			di.AsSelf[Issuer](),
			di.AsSelf[Validator](),
			di.AsSelf[Revoker](),
			di.AsSelf[Rotator](),
			di.AsSelf[MiddlewareFactory](),
		),
	}
	nodes = append(nodes, di.ConvertAnys(opts)...)
	return di.Options(nodes...)
}

type newJWTManagerIn struct {
	fx.In

	Config      jws.Config
	Signer      jws.SignerVerifier
	CacheStore  RevocationCache    `optional:"true"`
	CustomStore RevocationStore    `optional:"true"`
	ManagerOpts []JWTManagerOption `group:"us/session/jwt_manager_options"`
}

func newJWTManagerWithDefaultRevocation(in newJWTManagerIn) (*JWTManager, error) {
	manager, err := NewJWTManager(in.Config, in.Signer)
	if err != nil {
		return nil, err
	}

	for _, opt := range in.ManagerOpts {
		if opt != nil {
			opt(manager)
		}
	}

	if in.CustomStore != nil {
		manager.SetRevocationStore(in.CustomStore)
		return manager, nil
	}
	if in.CacheStore != nil {
		namespace := strings.TrimSpace(manager.config.Issuer)
		manager.SetRevocationStore(NewRevocationStoreWithNamespace(in.CacheStore, "", namespace))
	}
	return manager, nil
}
