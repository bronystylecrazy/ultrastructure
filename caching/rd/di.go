package rd

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.Module(
			"us/caching/redis",
			di.Config[Config]("caching.redis"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
			di.Provide(NewClient, interfaces()...),
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
