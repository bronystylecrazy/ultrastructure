package apikey

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func Providers(opts ...di.Node) di.Node {
	return di.Module(
		"us/apikey",
		cfg.Config[Config]("apikey", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		di.Provide(NewKeyGenerator, di.AsSelf[Generator]()),
		di.Provide(NewHasherFromConfig),
		di.Provide(func(in Params) *Service {
			return NewService(NewServiceParams{
				Config:    in.Config,
				Generator: in.Generator,
				Hasher:    in.Hasher,
				Lookup:    in.Lookup,
				Recorder:  in.Recorder,
				Revoker:   in.Revoker,
				Rotator:   in.Rotator,
			})
		}, di.AsSelf[Manager]()),
		di.Options(di.ConvertAnys(opts)...),
	)
}

type Params struct {
	fx.In

	Config    Config
	Generator Generator
	Hasher    Hasher
	Lookup    KeyLookup        `optional:"true"`
	Recorder  KeyUsageRecorder `optional:"true"`
	Revoker   Revoker          `optional:"true"`
	Rotator   Rotator          `optional:"true"`
}
