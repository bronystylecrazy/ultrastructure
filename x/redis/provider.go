package rd

import (
	"sync"

	"github.com/alicebob/miniredis/v2"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/redis/go-redis/extra/redisotel/v9"
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

// NewInstrumentedClient builds the client NewClient does and, when telemetry
// is enabled, traces its commands through tp and reports its pool and command
// metrics through mp. Spans are named after the command and leave out its
// arguments, which carry keys and values such as cached credentials. With
// telemetry disabled it builds the same client as NewClient.
func NewInstrumentedClient(cfg Config, otelConfig otel.Config, tp *otel.TracerProvider, mp *otel.MeterProvider) (*redis.Client, error) {
	client, err := NewClient(cfg)
	if err != nil || !otelConfig.Enabled {
		return client, err
	}
	if tp != nil {
		err = redisotel.InstrumentTracing(client,
			redisotel.WithTracerProvider(tp),
			redisotel.WithDBStatement(false),
		)
	}
	if err == nil && mp != nil {
		err = redisotel.InstrumentMetrics(client, redisotel.WithMeterProvider(mp))
	}
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
