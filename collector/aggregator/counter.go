package aggregator

import (
	"context"
	"hawkeye/quiver"
	"math"
	"time"
)

type Aggregator interface {
	Collect(ctx context.Context, metric string, interval time.Duration) float32
}

type CountAggregator struct {
	repo quiver.Repository
}

func NewCountAggregator(repo quiver.Repository) *CountAggregator {
	return &CountAggregator{repo: repo}
}

func (c *CountAggregator) Collect(ctx context.Context, metric string, interval time.Duration) float32 {
	value := c.repo.GetCountRange(ctx, metric, interval)
	return float32(math.Round(float64(value)))
}
