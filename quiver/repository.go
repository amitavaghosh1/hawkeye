package quiver

import (
	"context"
	"fmt"
	"hawkeye/utils"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	GetCountRange(ctx context.Context, metric string, interval time.Duration) float32
	SetCount(ctx context.Context, metric string, key int64, value float32) error
	DeleteCountRange(ctx context.Context, metric string, interval time.Duration) error
	// GetGauge()
	// SetGauge
}

type RedisRepo struct {
	client *redis.Client
}

const CounterCacheKeySuffix = "::timestamps"

func NewRedisRepo(client *redis.Client) *RedisRepo {
	return &RedisRepo{client: client}
}

// Here key should be time.Now().UTC().Unix()
func (rr *RedisRepo) SetCount(ctx context.Context, metric string, key int64, value float32) error {
	fv := fmt.Sprintf("%.3f", value)
	// strKey := fmt.Sprintf("%d", key)

	_, err := rr.client.
		HSet(ctx, metric, key, fv).
		Result()

	if err != nil {
		return err
	}

	_, err = rr.client.
		ZAdd(ctx, metric+CounterCacheKeySuffix, &redis.Z{Score: float64(key), Member: key}).
		Result()

	return err
}

func (rr *RedisRepo) GetCountRange(ctx context.Context, metric string, interval time.Duration) float32 {
	now := utils.Now()

	rang := &redis.ZRangeBy{
		Min: strconv.Itoa(int(utils.ToUnix(now.Add(-interval)))),
		Max: strconv.Itoa(int(utils.ToUnix(now))),
	}

	timestamps, err := rr.client.ZRangeByScoreWithScores(ctx, metric+CounterCacheKeySuffix, rang).Result()
	if err != nil {
		return 0
	}

	var count float64

	for _, timestamp := range timestamps {
		member, ok := timestamp.Member.(string)
		if !ok {
			log.Printf("invalid member type, expected int64 %#v\n", timestamp.Member)
			continue
		}

		val, err := rr.client.HGet(ctx, metric, member).Result()
		if err != nil {
			log.Println("value not set for key", metric, member, "err ", err)
			continue
		}

		if val == "" {
			log.Println("empty count value for", metric, member)
			continue
		}

		f, err := strconv.ParseFloat(val, 32)
		if err != nil {
			log.Println("failed to convert to float", metric, val)
			continue
		}

		// log.Println("metric value ", f)
		count += f
	}

	// log.Println("total count for", metric, count, " from ", len(timestamps))
	return float32(count)
}

func (rr *RedisRepo) DeleteCountRange(ctx context.Context, metric string, interval time.Duration) error {
	now := utils.Now()
	upto := strconv.Itoa(int(utils.ToUnix(now.Add(-interval))))

	rang := &redis.ZRangeBy{
		Max: upto,
		Min: "-inf",
	}

	key := metric + CounterCacheKeySuffix

	timestamps, err := rr.client.ZRangeByScoreWithScores(ctx, key, rang).Result()
	if err != nil {
		return nil
	}

	for _, timestamp := range timestamps {
		member, ok := timestamp.Member.(int64)
		if !ok {
			log.Println("invalid member type, expected int64")
			continue
		}

		_, err := rr.client.HDel(ctx, metric, strconv.Itoa(int(member))).Result()
		if err != nil {
			log.Println("failed to delete timestamp from hash", metric)
			continue
		}
	}

	_, err = rr.client.ZRemRangeByScore(ctx, key, "-inf", upto).Result()
	return err
}
