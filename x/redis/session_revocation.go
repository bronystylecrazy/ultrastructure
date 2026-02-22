package rd

import (
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/security/jws"
	"github.com/bronystylecrazy/ultrastructure/security/session"
)

type SessionRevocationOption func(*sessionRevocationOptions)

type sessionRevocationOptions struct {
	keyPrefix string
	namespace string
}

func WithSessionRevocationKeyPrefix(keyPrefix string) SessionRevocationOption {
	return func(o *sessionRevocationOptions) {
		o.keyPrefix = keyPrefix
	}
}

func WithSessionRevocationNamespace(namespace string) SessionRevocationOption {
	return func(o *sessionRevocationOptions) {
		o.namespace = namespace
	}
}

func UseSessionRevocation(opts ...SessionRevocationOption) di.Node {
	return di.Provide(func(config jws.Config, cacheStore session.RevocationCache) session.RevocationStore {
		if cacheStore == nil {
			return nil
		}
		options := sessionRevocationOptions{}
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

		return session.NewRevocationStoreWithNamespace(cacheStore, options.keyPrefix, namespace)
	}, di.AsSelf[session.RevocationStore](), di.Params(``, di.Optional()))
}

func UseSessionRevocationWithNamespace(keyPrefix string, namespace string) di.Node {
	return UseSessionRevocation(
		WithSessionRevocationKeyPrefix(keyPrefix),
		WithSessionRevocationNamespace(namespace),
	)
}
