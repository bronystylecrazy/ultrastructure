package spa

import (
	"embed"

	"go.uber.org/zap"
)

const DefaultDistPath = "web/dist"

type Option func(*Options)

type Options struct {
	assets   *embed.FS
	log      *zap.Logger
	distPath string
}

func WithAssets(assets *embed.FS) Option {
	return func(o *Options) {
		o.assets = assets
	}
}

func WithLogger(log *zap.Logger) Option {
	return func(o *Options) {
		o.log = log
	}
}

func WithDistPath(path string) Option {
	return func(o *Options) {
		o.distPath = path
	}
}
