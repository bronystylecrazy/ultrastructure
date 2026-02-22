package spa

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type Middleware struct {
	fs         fs.FS
	log        *zap.Logger
	indexPlain []byte
	indexGzip  []byte
}

func NewMiddleware(opts ...Option) (*Middleware, error) {
	cfg := Options{distPath: DefaultDistPath}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

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

	s.log.Debug("static middleware enabled")

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
