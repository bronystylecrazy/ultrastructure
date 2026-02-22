package token

import (
	"strings"

	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type ServiceOption func(*Service)

const serviceOptionsGroupName = "us/token/service_options"

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/token",
		cfg.Config[Config]("token", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		di.Provide(newServiceWithDefaultRevocation, di.AsSelf[Manager]()),
		di.Options(di.ConvertAnys(opts)...),
	)
}

type newServiceIn struct {
	fx.In

	Config      Config
	CacheStore  RevocationCache `optional:"true"`
	CustomStore RevocationStore `optional:"true"`
	ServiceOpts []ServiceOption `group:"us/token/service_options"`
}

func newServiceWithDefaultRevocation(in newServiceIn) (*Service, error) {
	service, err := NewService(in.Config)
	if err != nil {
		return nil, err
	}

	for _, opt := range in.ServiceOpts {
		if opt != nil {
			opt(service)
		}
	}

	if in.CustomStore != nil {
		service.SetRevocationStore(in.CustomStore)
		return service, nil
	}
	if in.CacheStore != nil {
		namespace := strings.TrimSpace(service.config.Issuer)
		service.SetRevocationStore(NewRedisRevocationStoreWithNamespace(in.CacheStore, "", namespace))
	}
	return service, nil
}
