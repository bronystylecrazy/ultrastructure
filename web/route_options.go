package web

// Name returns a RouteOption that sets operationId.
func Name(name string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Name(name)
	}
}

// TaggedName returns a RouteOption that sets operationId as <firstTag>_<name>.
func TaggedName(name string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.TaggedName(name)
	}
}

// Tag returns a RouteOption that adds one tag.
func Tag(tag string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Tags(tag)
	}
}

// Tags returns a RouteOption that adds multiple tags.
func Tags(tags ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Tags(tags...)
	}
}

// Security returns a RouteOption that sets a security scheme requirement without scopes.
func Security(scheme string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Security(scheme)
	}
}

// Scopes returns a RouteOption that sets a security scheme requirement with scopes.
func Scopes(scheme string, scopes ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Scopes(scheme, scopes...)
	}
}

// Policy returns a RouteOption that adds one policy requirement.
func Policy(name string) RouteOption {
	return Policies(name)
}

// Policies returns a RouteOption that adds policy requirements.
func Policies(names ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Policies(names...)
	}
}

// Public returns a RouteOption that marks the route as explicitly public.
func Public() RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Public()
	}
}
