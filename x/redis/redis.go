package rd

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

type ACLManager interface {
	redis.ACLCmdable
}

type BitMapManager interface {
	redis.BitMapCmdable
}

type ClusterManager interface {
	redis.ClusterCmdable
}

type GenericManager interface {
	redis.GenericCmdable
}

type GeoManager interface {
	redis.GeoCmdable
}

type HashManager interface {
	redis.HashCmdable
}

type HyperLogLogManager interface {
	redis.HyperLogLogCmdable
}

type ListManager interface {
	redis.ListCmdable
}

type ProbabilisticManager interface {
	redis.ProbabilisticCmdable
}

type PubSubManager interface {
	redis.PubSubCmdable
}

type ScriptingManager interface {
	redis.ScriptingFunctionsCmdable
}

type SearchManager interface {
	redis.SearchCmdable
}

type SetManager interface {
	redis.SetCmdable
}

type SortedSetManager interface {
	redis.SortedSetCmdable
}

type StringManager interface {
	redis.StringCmdable
}

type StreamManager interface {
	redis.StreamCmdable
}

type TimeseriesManager interface {
	redis.TimeseriesCmdable
}

type JSONManager interface {
	redis.JSONCmdable
}

type VectorSetManager interface {
	redis.VectorSetCmdable
}

type HookAdder interface {
	AddHook(hook redis.Hook)
}

type Watcher interface {
	Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error
}

type Processor interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
	Process(ctx context.Context, cmd redis.Cmder) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
	PSubscribe(ctx context.Context, channels ...string) *redis.PubSub
	SSubscribe(ctx context.Context, channels ...string) *redis.PubSub
}

type Closer interface {
	Close() error
}

type PoolStatser interface {
	PoolStats() *redis.PoolStats
}

type Commander interface {
	redis.Cmdable
}

type StatefulCommander interface {
	redis.StatefulCmdable
}

type RedisManager interface {
	Commander
	ACLManager
	BitMapManager
	ClusterManager
	GenericManager
	GeoManager
	HashManager
	HyperLogLogManager
	ListManager
	ProbabilisticManager
	PubSubManager
	ScriptingManager
	SearchManager
	SetManager
	SortedSetManager
	StringManager
	StreamManager
	TimeseriesManager
	JSONManager
	VectorSetManager
}

type RedisClient interface {
	RedisManager
	HookAdder
	Watcher
	Processor
	Subscriber
	Closer
	PoolStatser
}
