package di

// AsSelf creates a new Option that provides the given value as a dependency and tags it with the given tags.
// Example usage:
//
//	di.Provide(NewHandler, di.AsSelf[realtime.Authorizer]())),
//
// Equivalent to:
//
//	di.Provide(NewHandler, di.As[realtime.Authorizer](), di.Self()),
func AsSelf[T any](tags ...string) Option {
	return Options(
		As[T](tags...),
		Self(),
	)
}
