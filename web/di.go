package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func IgnoreAutoGroupHandlers() di.Option {
	return di.AutoGroupIgnoreType[Handler](HandlersGroupName)
}

func Init() di.Node {
	return di.Options(
		di.AutoGroup[Handler](HandlersGroupName),
		di.Provide(NewServeCommand),
		di.Invoke(SetupHandlers, Priority(Earlier)),
		di.Invoke(func(s Server) error {
			errCh := make(chan error, 1)
			go func() {
				errCh <- s.Listen()
			}()

			select {
			case <-s.Wait():
				return nil
			case err := <-errCh:
				return err
			}
		}),
	)
}

// UseServeCommand is no longer needed: Init registers the serve command.
//
// Deprecated: web.Init provides the serve command automatically.
func UseServeCommand() di.Node {
	return di.Options()
}

func UseBuildInfo(opts ...BuildInfoOption) di.Node {
	return di.Options(
		di.Provide(func() *BuildInfoHandler {
			return NewBuildInfoHandler(opts...)
		}),
	)
}
