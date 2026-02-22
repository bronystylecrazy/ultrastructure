package rd_test

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/x/redis"
	redis "github.com/redis/go-redis/v9"
)

var _ rd.ACLManager = (*redis.Client)(nil)
var _ rd.BitMapManager = (*redis.Client)(nil)
var _ rd.ClusterManager = (*redis.Client)(nil)
var _ rd.GenericManager = (*redis.Client)(nil)
var _ rd.GeoManager = (*redis.Client)(nil)
var _ rd.HashManager = (*redis.Client)(nil)
var _ rd.HyperLogLogManager = (*redis.Client)(nil)
var _ rd.ListManager = (*redis.Client)(nil)
var _ rd.ProbabilisticManager = (*redis.Client)(nil)
var _ rd.PubSubManager = (*redis.Client)(nil)
var _ rd.ScriptingManager = (*redis.Client)(nil)
var _ rd.SearchManager = (*redis.Client)(nil)
var _ rd.SetManager = (*redis.Client)(nil)
var _ rd.SortedSetManager = (*redis.Client)(nil)
var _ rd.StringManager = (*redis.Client)(nil)
var _ rd.StreamManager = (*redis.Client)(nil)
var _ rd.TimeseriesManager = (*redis.Client)(nil)
var _ rd.JSONManager = (*redis.Client)(nil)
var _ rd.VectorSetManager = (*redis.Client)(nil)

var _ rd.HookAdder = (*redis.Client)(nil)
var _ rd.Watcher = (*redis.Client)(nil)
var _ rd.Processor = (*redis.Client)(nil)
var _ rd.Subscriber = (*redis.Client)(nil)
var _ rd.Closer = (*redis.Client)(nil)
var _ rd.PoolStatser = (*redis.Client)(nil)

var _ rd.Commander = (*redis.Client)(nil)
var _ rd.StatefulCommander = (*redis.Conn)(nil)

var _ rd.RedisManager = (*redis.Client)(nil)

var _ rd.RedisClient = (*redis.Client)(nil)
var _ rd.RedisClient = (*redis.ClusterClient)(nil)
var _ rd.RedisClient = (*redis.Ring)(nil)

func TestNewRedisClientInMemory(t *testing.T) {
	client, err := rd.NewClient(rd.Config{InMemory: true})
	if err != nil {
		t.Fatalf("new redis client: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Fatalf("close redis client: %v", closeErr)
		}
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}

	if err := client.Set(ctx, "k", "v", 0).Err(); err != nil {
		t.Fatalf("set redis key: %v", err)
	}

	got, err := client.Get(ctx, "k").Result()
	if err != nil {
		t.Fatalf("get redis key: %v", err)
	}
	if got != "v" {
		t.Fatalf("unexpected value: got %q want %q", got, "v")
	}
}
