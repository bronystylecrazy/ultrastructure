package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web/buildinfo"
	"go.uber.org/zap"
)

type BuildInfoOption = buildinfo.Option

type BuildInfoHandler = buildinfo.Handler

func UseBuildInfo(opts ...BuildInfoOption) di.Node {
	return di.Options(
		di.Provide(func() *buildinfo.Handler {
			return buildinfo.NewHandler(opts...)
		}),
		di.Invoke(func(log *zap.Logger) {
			log.Debug("use build info handler")
		}),
	)
}

func WithDefaultPath(path ...string) BuildInfoOption {
	return buildinfo.WithDefaultPath(path...)
}

func NewBuildInfoHandler(opts ...BuildInfoOption) *BuildInfoHandler {
	return buildinfo.NewHandler(opts...)
}
