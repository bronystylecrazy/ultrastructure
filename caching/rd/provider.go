package rd

import (
	"sync"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

var (
	inMemoryRedisMu     sync.Mutex
	inMemoryRedisServer *miniredis.Miniredis
)

func NewClient(cfg Config) (*redis.Client, error) {
	options := cfg.Options()
	if cfg.InMemory {
		addr, err := ensureInMemoryRedisAddr()
		if err != nil {
			return nil, err
		}
		options.Addr = addr
	}

	return redis.NewClient(options), nil
}

func ensureInMemoryRedisAddr() (string, error) {
	inMemoryRedisMu.Lock()
	defer inMemoryRedisMu.Unlock()

	if inMemoryRedisServer != nil {
		return inMemoryRedisServer.Addr(), nil
	}

	server, err := miniredis.Run()
	if err != nil {
		return "", err
	}

	inMemoryRedisServer = server
	return inMemoryRedisServer.Addr(), nil
}
