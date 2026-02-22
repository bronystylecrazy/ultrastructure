package goose

import (
	"go.uber.org/zap"
)

type Option interface {
	apply(*option)
}

type option struct {
	paths  []string
	logger *zap.Logger
}

type optionFunc func(*option)

func (f optionFunc) apply(cfg *option) {
	f(cfg)
}

func WithPath(path string) Option {
	return optionFunc(func(cfg *option) {
		cfg.paths = append(cfg.paths, path)
	})
}

func WithPaths(paths ...string) Option {
	return optionFunc(func(cfg *option) {
		cfg.paths = append(cfg.paths, paths...)
	})
}

func WithLogger(logger *zap.Logger) Option {
	return optionFunc(func(cfg *option) {
		cfg.logger = logger
	})
}
