package rd_test

import (
	"context"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/caching/rd"
	"github.com/bronystylecrazy/ultrastructure/di"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func TestUseInterfacesProvidesRedisInterfaces(t *testing.T) {
	var raw *redis.Client
	var client rd.RedisClient
	var manager rd.RedisManager
	var strings rd.StringManager
	var closer rd.Closer

	app := fx.New(
		di.App(
			di.Supply(rd.Config{InMemory: true}),
			di.Provide(rd.NewClient),
			rd.UseInterfaces(),
			di.Populate(&raw),
			di.Populate(&client),
			di.Populate(&manager),
			di.Populate(&strings),
			di.Populate(&closer),
		).Build(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start app: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if raw == nil {
		t.Fatal("raw redis client is nil")
	}
	if client == nil {
		t.Fatal("redis client interface is nil")
	}
	if manager == nil {
		t.Fatal("redis manager interface is nil")
	}
	if strings == nil {
		t.Fatal("string manager interface is nil")
	}
	if closer == nil {
		t.Fatal("closer interface is nil")
	}

	if got := any(client).(*redis.Client); got != raw {
		t.Fatal("redis client interface does not point to the same client instance")
	}

	if err := raw.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
}
