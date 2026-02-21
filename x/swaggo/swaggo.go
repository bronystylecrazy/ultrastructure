package swaggo

import (
	"github.com/Flussen/swagger-fiber-v3"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/bronystylecrazy/ultrastructure/x/autoswag"
)

type Option = autoswag.Option
type Context = autoswag.Context
type HookFunc = autoswag.HookFunc

type Middleware struct {
	path string
}

func Use(opts ...Option) di.Node {
	return di.Options(
		di.Provide(func() (*Middleware, error) {
			resolved := autoswag.ResolveOptions("/docs/*", opts...)
			return &Middleware{path: resolved.Path}, nil
		}, di.AutoGroupIgnoreType[web.Handler](web.HandlersGroupName)),
		di.Invoke(func(router web.Router, swaggerMiddleware *Middleware) {
			if swaggerMiddleware == nil {
				return
			}
			swaggerMiddleware.Handle(router)
		}, di.Params(di.Optional(), di.Optional())),
	)
}

func (m *Middleware) Handle(r web.Router) {
	r.Get(m.path, swagger.HandlerDefault)
}

func WithConfig(config web.Config) Option {
	return autoswag.WithConfig(config)
}

func WithPath(path string) Option {
	return autoswag.WithPath(path)
}

func WithEmitFiles(paths ...string) Option {
	return autoswag.WithEmitFiles(paths...)
}

func WithEmitFile(path string) Option {
	return autoswag.WithEmitFile(path)
}

func WithVersionedDocs(path, prefix, name string) Option {
	return autoswag.WithVersionedDocs(path, prefix, name)
}

func WithBearerSecurityScheme(name string) Option {
	return autoswag.WithBearerSecurityScheme(name)
}

func WithAPIKeySecurityScheme(name, keyName, in string) Option {
	return autoswag.WithAPIKeySecurityScheme(name, keyName, in)
}

func WithDefaultSecurity(scheme string, scopes ...string) Option {
	return autoswag.WithDefaultSecurity(scheme, scopes...)
}

func WithTagDescription(name, description string) Option {
	return autoswag.WithTagDescription(name, description)
}

func WithPackageTagTransform(transform func(string) string) Option {
	return autoswag.WithPackageTagTransform(transform)
}

func WithTermsOfService(url string) Option {
	return autoswag.WithTermsOfService(url)
}

func WithContact(name, url, email string) Option {
	return autoswag.WithContact(name, url, email)
}

func WithLicense(name, url string) Option {
	return autoswag.WithLicense(name, url)
}

func WithCustomize(hook HookFunc) Option {
	return autoswag.WithCustomize(hook)
}

func WitHook(hook ...HookFunc) Option {
	return autoswag.WitHook(hook...)
}

func WithExtraModels(models ...any) Option {
	return autoswag.WithExtraModels(models...)
}
