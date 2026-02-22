package token

import (
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
)

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

func UseRefreshRoute(path string, opts ...RefreshRouteOption) di.Node {
	return di.Provide(func(service Manager) *RefreshHandler {
		h := NewRefreshHandler(service).
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

func UseDefaultAccessExtractors(exs ...Extractor) di.Node {
	return di.Provide(func() ServiceOption {
		return func(service *Service) {
			service.SetDefaultAccessExtractors(exs...)
		}
	}, di.Group(serviceOptionsGroupName))
}

func UseDefaultRefreshExtractors(exs ...Extractor) di.Node {
	return di.Provide(func() ServiceOption {
		return func(service *Service) {
			service.SetDefaultRefreshExtractors(exs...)
		}
	}, di.Group(serviceOptionsGroupName))
}

func UseRevocationStore(store RevocationStore) di.Node {
	return di.Supply(store, di.AsSelf[RevocationStore]())
}

type RedisRevocationOption func(*redisRevocationOptions)

type redisRevocationOptions struct {
	keyPrefix string
	namespace string
}

func WithRedisRevocationKeyPrefix(keyPrefix string) RedisRevocationOption {
	return func(o *redisRevocationOptions) {
		o.keyPrefix = keyPrefix
	}
}

func WithRedisRevocationNamespace(namespace string) RedisRevocationOption {
	return func(o *redisRevocationOptions) {
		o.namespace = namespace
	}
}

func UseRedisRevocation(opts ...RedisRevocationOption) di.Node {
	return di.Provide(func(config Config, cacheStore RevocationCache) RevocationStore {
		if cacheStore == nil {
			return nil
		}
		options := redisRevocationOptions{}
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			opt(&options)
		}

		namespace := strings.TrimSpace(options.namespace)
		if namespace == "" {
			namespace = strings.TrimSpace(config.Issuer)
		}

		return NewRedisRevocationStoreWithNamespace(cacheStore, options.keyPrefix, namespace)
	}, di.AsSelf[RevocationStore](), di.Params(``, di.Optional()))
}

func UseRedisRevocationWithNamespace(keyPrefix string, namespace string) di.Node {
	return UseRedisRevocation(
		WithRedisRevocationKeyPrefix(keyPrefix),
		WithRedisRevocationNamespace(namespace),
	)
}
