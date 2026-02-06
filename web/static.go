package web

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

const DefaultSpaDistPath = "web/dist"

type SpaOption func(*spaOptions)

type SpaMiddleware struct {
	fs         fs.FS
	log        *zap.Logger
	indexPlain []byte
	indexGzip  []byte
}

type spaOptions struct {
	assets   *embed.FS
	log      *zap.Logger
	distPath string
}

func WithSpaAssets(assets *embed.FS) SpaOption {
	return func(o *spaOptions) {
		o.assets = assets
	}
}

func WithSpaLogger(log *zap.Logger) SpaOption {
	return func(o *spaOptions) {
		o.log = log
	}
}

func WithSpaDistPath(path string) SpaOption {
	return func(o *spaOptions) {
		o.distPath = path
	}
}

func NewSpaMiddleware(assets *embed.FS, log *zap.Logger) (*SpaMiddleware, error) {
	return NewSpaMiddlewareWithOptions(WithSpaAssets(assets), WithSpaLogger(log))
}

func NewSpaMiddlewareWithOptions(opts ...SpaOption) (*SpaMiddleware, error) {
	cfg := spaOptions{distPath: DefaultSpaDistPath}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return newSpaMiddlewareWithOptions(cfg)
}

func newSpaMiddlewareWithOptions(cfg spaOptions) (*SpaMiddleware, error) {
	if cfg.log == nil {
		cfg.log = zap.NewNop()
	}
	if cfg.assets == nil {
		return &SpaMiddleware{fs: nil, log: cfg.log}, nil
	}

	subbedDist, err := fs.Sub(cfg.assets, cfg.distPath)
	if err != nil {
		return nil, err
	}

	return &SpaMiddleware{fs: subbedDist, log: cfg.log}, nil
}

func (s *SpaMiddleware) Handle(r fiber.Router) {
	if s.fs == nil {
		s.log.Debug("skip static middleware, assets(*embed.FS) is not provided/supplied")
		return
	}

	r.Use(func(c fiber.Ctx) error {
		path := c.Path()
		if path == "/" {
			path = "/index.html"
		}

		// Check gzip support
		acceptsGzip := strings.Contains(c.Get("Accept-Encoding"), "gzip")

		// Try to serve .gz version
		if acceptsGzip {
			gzPath := path + ".gz"
			if file, err := s.fs.Open(gzPath[1:]); err == nil {
				defer file.Close()
				c.Set("Content-Encoding", "gzip")
				c.Set("Vary", "Accept-Encoding")
				c.Type(filepath.Ext(path)) // get mime based on original ext
				return c.SendStream(file)
			}
		}

		// Try to serve normal file
		file, err := s.fs.Open(path[1:])
		if err != nil {
			// Fallback to index.html (SPA)
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
