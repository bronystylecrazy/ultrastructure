package paseto

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
)

// Providers returns a DI node that provides PASETO SignerVerifier.
func Providers() di.Node {
	return di.Options(
		cfg.Config[Config]("paseto", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		di.Provide(New, di.AsSelf[Signer](), di.AsSelf[Verifier](), di.AsSelf[SignerVerifier]()),
	)
}

// UseConfig supplies a PASETO config directly.
func UseConfig(cfg Config) di.Node {
	return di.Supply(cfg, di.AsSelf[Config]())
}

// UseSignerVerifier supplies an existing SignerVerifier.
func UseSignerVerifier(sv SignerVerifier) di.Node {
	return di.Options(
		di.Supply(sv, di.AsSelf[Signer](), di.AsSelf[Verifier](), di.AsSelf[SignerVerifier]()),
	)
}

// UseSigner supplies an existing Signer.
func UseSigner(signer Signer) di.Node {
	return di.Supply(signer, di.AsSelf[Signer]())
}

// UseVerifier supplies an existing Verifier.
func UseVerifier(verifier Verifier) di.Node {
	return di.Supply(verifier, di.AsSelf[Verifier]())
}
