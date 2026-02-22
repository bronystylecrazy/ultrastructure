package rd

import (
	"context"
	"errors"
	"time"

	"github.com/bronystylecrazy/ultrastructure/security/apikey"
	redis "github.com/redis/go-redis/v9"
)

type apikeyCacheStoreAdapter struct {
	client *redis.Client
}

func NewAPIKeyCacheStore(client *redis.Client) apikey.CacheStore {
	if client == nil {
		return nil
	}
	return apikeyCacheStoreAdapter{client: client}
}

func (a apikeyCacheStoreAdapter) Get(ctx context.Context, key string) (string, error) {
	val, err := a.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", apikey.ErrCacheMiss
		}
		return "", err
	}
	return val, nil
}

func (a apikeyCacheStoreAdapter) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return a.client.Set(ctx, key, value, ttl).Err()
}

func (a apikeyCacheStoreAdapter) Del(ctx context.Context, keys ...string) error {
	return a.client.Del(ctx, keys...).Err()
}
