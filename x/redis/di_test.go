package rd_test

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	"github.com/bronystylecrazy/ultrastructure/x/redis"
	redis "github.com/redis/go-redis/v9"
)

func TestUseInterfacesProvidesRedisInterfaces(t *testing.T) {
	var raw *redis.Client
	var client rd.RedisClient
	var manager rd.RedisManager
	var strings rd.StringManager
	var closer rd.Closer

	defer ustest.New(
		t,
		rd.Providers(),
		di.Populate(&raw),
		di.Populate(&client),
		di.Populate(&manager),
		di.Populate(&strings),
		di.Populate(&closer),
	).RequireStart().RequireStop()

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
}
