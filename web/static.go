package web

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func UseSpa(assets *embed.FS) di.Node {
	return di.Options(
		di.Supply(assets, di.Private()),
		di.Provide(NewSpaMiddleware, di.Params(di.Optional()), Priority(Latest)),
	)
}

func HandleSpa(r fiber.Router, s *SpaMiddleware) {
	s.Handle(r)
}

func NopSpa() di.Node {
	return di.Replace(&SpaMiddleware{fs: nil, log: zap.NewNop()})
}

type SpaMiddleware struct {
	fs         fs.FS
	log        *zap.Logger
	indexPlain []byte
	indexGzip  []byte
}

func NewSpaMiddleware(assets *embed.FS, log *zap.Logger) (*SpaMiddleware, error) {
	if assets == nil {
		return &SpaMiddleware{fs: nil, log: log}, nil
	}

	subbedDist, err := fs.Sub(assets, "web/dist")
	if err != nil {
		return nil, err
	}

	return &SpaMiddleware{fs: subbedDist, log: log}, nil
}

func (s *SpaMiddleware) Handle(r fiber.Router) {
	s.log.Info("initialize static middleware")
	if s.fs == nil {
		s.log.Info("skip static middleware, assets(*embed.FS) is not provided/supplied")
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
