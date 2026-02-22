package session

import "github.com/bronystylecrazy/ultrastructure/di"

type JWTManagerOption func(*JWTManager)

const JWTManagerOptionsGroupName = "us/session/jwt_manager_options"

func UseJWTManagerOptions(opts ...JWTManagerOption) di.Node {
	nodes := make([]any, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		option := opt
		nodes = append(nodes, di.Provide(func() JWTManagerOption {
			return option
		}, di.Group(JWTManagerOptionsGroupName)))
	}
	return di.Options(nodes...)
}

func WithAccessExtractors(extractors ...Extractor) JWTManagerOption {
	return func(m *JWTManager) {
		m.SetDefaultAccessExtractors(extractors...)
	}
}

func WithRefreshExtractors(extractors ...Extractor) JWTManagerOption {
	return func(m *JWTManager) {
		m.SetDefaultRefreshExtractors(extractors...)
	}
}

type RefreshRouteOption func(*RefreshHandler)

func WithRefreshPairDeliverer(deliverer PairDeliverer) RefreshRouteOption {
	return func(h *RefreshHandler) {
		h.WithDeliverer(deliverer)
	}
}

func WithRefreshPairDelivererResolver(resolver PairDelivererResolver) RefreshRouteOption {
	return func(h *RefreshHandler) {
		h.WithDelivererResolver(resolver)
	}
}

func WithRefreshSubjectResolver(subjectResolver SubjectResolver) RefreshRouteOption {
	return func(h *RefreshHandler) {
		h.WithSubjectResolver(subjectResolver)
	}
}

func UseRefreshRoute(path string, opts ...RefreshRouteOption) di.Node {
	return di.Provide(func(service Manager) *RefreshHandler {
		h := NewRefreshHandler(service, nil).
			WithPath(path).
			WithDelivererResolver(WebCookieOrJSONPairDelivererResolver())
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			opt(h)
		}
		return h
	})
}

func UseRevocationStore(store RevocationStore) di.Node {
	return di.Supply(store, di.AsSelf[RevocationStore]())
}
