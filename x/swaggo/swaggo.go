package swaggo

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/x/autoswag"
)

func Use(opts ...Option) di.Node {
	return di.Options(
		di.Provide(func() (*Middleware, error) {
			return &Middleware{path: autoswag.ResolveOptions("/docs/*", opts...).Path}, nil
		}),
	)
}
