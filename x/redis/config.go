package rd

import (
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Config struct {
	InMemory        bool          `mapstructure:"in_memory" default:"false"`
	Network         string        `mapstructure:"network" default:"tcp"`
	Addr            string        `mapstructure:"addr" default:"127.0.0.1:6379"`
	Protocol        int           `mapstructure:"protocol" default:"3"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db" default:"0"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout" default:"5s"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" default:"3s"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" default:"3s"`
	PoolSize        int           `mapstructure:"pool_size" default:"10"`
	MinIdleConns    int           `mapstructure:"min_idle_conns" default:"0"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"0"`
	MaxActiveConns  int           `mapstructure:"max_active_conns" default:"0"`
	PoolTimeout     time.Duration `mapstructure:"pool_timeout" default:"4s"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" default:"30m"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"0s"`
}

func (c Config) Options() *redis.Options {
	return &redis.Options{
		Network:         c.Network,
		Addr:            c.Addr,
		Protocol:        c.Protocol,
		Username:        c.Username,
		Password:        c.Password,
		DB:              c.DB,
		DialTimeout:     c.DialTimeout,
		ReadTimeout:     c.ReadTimeout,
		WriteTimeout:    c.WriteTimeout,
		PoolSize:        c.PoolSize,
		MinIdleConns:    c.MinIdleConns,
		MaxIdleConns:    c.MaxIdleConns,
		MaxActiveConns:  c.MaxActiveConns,
		PoolTimeout:     c.PoolTimeout,
		ConnMaxIdleTime: c.ConnMaxIdleTime,
		ConnMaxLifetime: c.ConnMaxLifetime,
	}
}

func NewOptions(cfg Config) *redis.Options {
	return cfg.Options()
}
