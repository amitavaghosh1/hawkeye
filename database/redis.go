package database

import (
	"context"
	"crypto/tls"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	localHost = "localhost"
	localIP   = "127.0.0"
)

var DefaultTlsConfig = &tls.Config{
	InsecureSkipVerify: true,
}

type RedisConfig struct {
	Addr         string
	DB           int
	MinIdleConns int
	PoolSize     int
	ReadTimeout  *time.Duration
	MaxRetries   *int
}

func ShouldUseTLS(url string) bool {
	isLocal := strings.HasPrefix(url, localHost) || strings.HasPrefix(url, localIP)

	return !isLocal
}

func NewRedisConnection(config RedisConfig, tlsCfg *tls.Config) *redis.Client {
	maxRetries := 2
	rconfig := redis.Options{
		Addr:         config.Addr,
		DB:           config.DB,
		MinIdleConns: 10,
		PoolSize:     20,
		MaxRetries:   maxRetries,
	}

	if ShouldUseTLS(config.Addr) {
		rconfig.TLSConfig = tlsCfg
	}

	if config.MinIdleConns > 0 {
		rconfig.MinIdleConns = config.MinIdleConns
	}

	if config.PoolSize > 0 {
		rconfig.PoolSize = config.PoolSize
	}

	if config.ReadTimeout != nil {
		rconfig.ReadTimeout = *config.ReadTimeout
	}

	if config.MaxRetries != nil {
		rconfig.MaxRetries = *config.MaxRetries
	}

	return redis.NewClient(&rconfig)
}

func NewRedisClient(url string) *redis.Client {
	client := NewRedisConnection(RedisConfig{
		Addr: url,
		DB:   0,
	}, DefaultTlsConfig)

	ctx := context.Background()

	// ctx, cancel := context.WithTimeout(
	// 	context.Background(),
	// 	1*time.Second,
	// )
	// defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal(err)
	}

	return client
}
