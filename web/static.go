package web

import (
	"embed"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type FS embed.FS

type StaticHandler interface {
	Handler
}

var NopStaticHandler StaticHandler = &NopHandler{}

type staticHandler struct {
	fs  FS
	log *zap.Logger
}

func NewStaticHandler(assets FS, log *zap.Logger) StaticHandler {
	return &staticHandler{fs: assets, log: log}
}

func (s *staticHandler) Handle(app App) {
	subbedDist, err := fs.Sub(embed.FS(s.fs), "web/dist")
	if err != nil {
		s.log.Fatal("failed to get subbed dist", zap.Error(err))
	}

	app.Use(func(c *fiber.Ctx) error {
		path := c.Path()
		if path == "/" {
			path = "/index.html"
		}

		// Check gzip support
		acceptsGzip := strings.Contains(c.Get("Accept-Encoding"), "gzip")

		// Try to serve .gz version
		if acceptsGzip {
			gzPath := path + ".gz"
			if file, err := subbedDist.Open(gzPath[1:]); err == nil {
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					return err
				}

				c.Set("Content-Encoding", "gzip")
				c.Set("Vary", "Accept-Encoding")
				c.Type(filepath.Ext(path)) // get mime based on original ext
				return c.Send(data)
			}
		}

		// Try to serve normal file
		file, err := subbedDist.Open(path[1:])
		if err != nil {
			// Fallback to index.html (SPA)
			index, err := subbedDist.Open("index.html.gz")
			if err != nil {
				return fiber.ErrNotFound
			}
			defer index.Close()
			data, err := io.ReadAll(index)
			if err != nil {
				return err
			}
			c.Set("Content-Encoding", "gzip")
			c.Set("Vary", "Accept-Encoding")
			c.Type("html")
			return c.Send(data)
		}

		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		c.Type(filepath.Ext(path))
		return c.Send(data)
	})
}
