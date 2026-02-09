package rd

import (
	redis "github.com/redis/go-redis/v9"

	"github.com/bronystylecrazy/ultrastructure/di"
)

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.Module(
			"us/caching/redis",
			di.Config[Config]("caching.redis"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
			di.Provide(NewClient),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}

func UseInterfaces() di.Node {
	return di.Provide(
		func(c *redis.Client) *redis.Client {
			return c
		},
		AsRedisClientInterfaces()...,
	)
}

func AsRedisClientInterfaces() []any {
	return []any{
		di.As[ACLManager](),
		di.As[BitMapManager](),
		di.As[ClusterManager](),
		di.As[GenericManager](),
		di.As[GeoManager](),
		di.As[HashManager](),
		di.As[HyperLogLogManager](),
		di.As[ListManager](),
		di.As[ProbabilisticManager](),
		di.As[PubSubManager](),
		di.As[ScriptingManager](),
		di.As[SearchManager](),
		di.As[SetManager](),
		di.As[SortedSetManager](),
		di.As[StringManager](),
		di.As[StreamManager](),
		di.As[TimeseriesManager](),
		di.As[JSONManager](),
		di.As[VectorSetManager](),
		di.As[HookAdder](),
		di.As[Watcher](),
		di.As[Processor](),
		di.As[Subscriber](),
		di.As[Closer](),
		di.As[PoolStatser](),
		di.As[Commander](),
		di.As[RedisManager](),
		di.As[RedisClient](),
	}
}
