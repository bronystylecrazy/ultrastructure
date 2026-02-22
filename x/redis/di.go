package rd

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/security/apikey"
	"github.com/bronystylecrazy/ultrastructure/security/token"
)

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.Module(
			"us/caching/redis",
			cfg.Config[Config]("caching.redis", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
			di.Provide(NewClient, interfaces()...),
			di.Provide(NewAPIKeyCacheStore, di.As[apikey.CacheStore]()),
			di.Provide(NewTokenRevocationCache, di.As[token.RevocationCache]()),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}

func interfaces() []any {
	return []any{
		di.AsSelf[ACLManager](),
		di.AsSelf[BitMapManager](),
		di.AsSelf[ClusterManager](),
		di.AsSelf[GenericManager](),
		di.AsSelf[GeoManager](),
		di.AsSelf[HashManager](),
		di.AsSelf[HyperLogLogManager](),
		di.AsSelf[ListManager](),
		di.AsSelf[ProbabilisticManager](),
		di.AsSelf[PubSubManager](),
		di.AsSelf[ScriptingManager](),
		di.AsSelf[SearchManager](),
		di.AsSelf[SetManager](),
		di.AsSelf[SortedSetManager](),
		di.AsSelf[StringManager](),
		di.AsSelf[StreamManager](),
		di.AsSelf[TimeseriesManager](),
		di.AsSelf[JSONManager](),
		di.AsSelf[VectorSetManager](),
		di.AsSelf[HookAdder](),
		di.AsSelf[Watcher](),
		di.AsSelf[Processor](),
		di.AsSelf[Subscriber](),
		di.AsSelf[Closer](),
		di.AsSelf[PoolStatser](),
		di.AsSelf[Commander](),
		di.AsSelf[RedisManager](),
		di.AsSelf[RedisClient](),
	}
}
