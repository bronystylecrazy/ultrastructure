package spa

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

const DefaultDistPath = "web/dist"

type Option func(*options)

type Middleware struct {
	fs         fs.FS
	log        *zap.Logger
	indexPlain []byte
	indexGzip  []byte
}

type options struct {
	assets   *embed.FS
	log      *zap.Logger
	distPath string
}

func Use(opts ...Option) di.Node {
	return di.Options(
		di.Provide(func(assets *embed.FS, log *zap.Logger) (*Middleware, error) {
			base := []Option{WithAssets(assets), WithLogger(log)}
			return NewMiddlewareWithOptions(append(base, opts...)...)
		}, di.AutoGroupIgnoreType[web.Handler](web.HandlersGroupName), di.Params(di.Optional()), web.Priority(web.Latest)),
		di.Invoke(func(router web.Router, spaMiddleware *Middleware) {
			if spaMiddleware == nil {
				return
			}
			spaMiddleware.Handle(router)
		}, di.Params(di.Optional(), di.Optional())),
	)
}

func WithAssets(assets *embed.FS) Option {
	return func(o *options) {
		o.assets = assets
	}
}

func WithLogger(log *zap.Logger) Option {
	return func(o *options) {
		o.log = log
	}
}

func WithDistPath(path string) Option {
	return func(o *options) {
		o.distPath = path
	}
}

func NewMiddleware(assets *embed.FS, log *zap.Logger) (*Middleware, error) {
	return NewMiddlewareWithOptions(WithAssets(assets), WithLogger(log))
}

func NewMiddlewareWithOptions(opts ...Option) (*Middleware, error) {
	cfg := options{distPath: DefaultDistPath}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return newMiddlewareWithOptions(cfg)
}

func newMiddlewareWithOptions(cfg options) (*Middleware, error) {
	if cfg.log == nil {
		cfg.log = zap.NewNop()
	}
	if cfg.assets == nil {
		return &Middleware{fs: nil, log: cfg.log}, nil
	}

	subbedDist, err := fs.Sub(cfg.assets, cfg.distPath)
	if err != nil {
		return nil, err
	}

	return &Middleware{fs: subbedDist, log: cfg.log}, nil
}

func (s *Middleware) Handle(r web.Router) {
	if s.fs == nil {
		s.log.Debug("skip static middleware, assets(*embed.FS) is not provided/supplied")
		return
	}

	r.Use(func(c fiber.Ctx) error {
		path := c.Path()
		if path == "/" {
			path = "/index.html"
		}

		acceptsGzip := strings.Contains(c.Get("Accept-Encoding"), "gzip")
		if acceptsGzip {
			gzPath := path + ".gz"
			if file, err := s.fs.Open(gzPath[1:]); err == nil {
				defer file.Close()
				c.Set("Content-Encoding", "gzip")
				c.Set("Vary", "Accept-Encoding")
				c.Type(filepath.Ext(path))
				return c.SendStream(file)
			}
		}

		file, err := s.fs.Open(path[1:])
		if err != nil {
			if acceptsGzip {
				if index, err := s.fs.Open("index.html.gz"); err == nil {
					defer index.Close()
					c.Set("Content-Encoding", "gzip")
					c.Set("Vary", "Accept-Encoding")
					c.Type("html")
					return c.SendStream(index)
				}
			}
			if index, err := s.fs.Open("index.html"); err == nil {
				defer index.Close()
				c.Type("html")
				return c.SendStream(index)
			}
			if acceptsGzip && len(s.indexGzip) > 0 {
				c.Set("Content-Encoding", "gzip")
				c.Set("Vary", "Accept-Encoding")
				c.Type("html")
				return c.Send(s.indexGzip)
			}
			if len(s.indexPlain) > 0 {
				c.Type("html")
				return c.Send(s.indexPlain)
			}
			return fiber.ErrNotFound
		}

		defer file.Close()
		c.Type(filepath.Ext(path))
		return c.SendStream(file)
	})
}
