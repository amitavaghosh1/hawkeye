package instruments

import (
	"context"
	"fmt"
	"hawkeye/collector/raider"
	"hawkeye/config"
	"hawkeye/utils"
	"log"
	"sync"
)

type Instrument struct {
	client utils.RPCClient
}

var (
	mu   sync.RWMutex
	inst *Instrument
)

func InstrumentWithConfig(cfg config.AppConfig) {
	mu.Lock()
	defer mu.Unlock()

	inst = &Instrument{
		client: utils.InitClientUnix(),
	}
}

func (i *Instrument) SendMetric(metric string) {
	var reply int
	i.client.Go("Metric.Handle", raider.Metric{Text: metric}, &reply, nil)
}

func (i *Instrument) Count(ctx context.Context, metric string, dir int) {
	value := 1 * dir
	mKey := fmt.Sprintf("%s:%d|c", metric, value)

	log.Println(mKey)

	i.SendMetric(mKey)
}

func (i *Instrument) Incr(ctx context.Context, metric string) {
	i.Count(ctx, metric, 1)

}

func (i *Instrument) Decr(ctx context.Context, metric string) {
	i.Count(ctx, metric, -1)
}

func Incr(ctx context.Context, metric string) {
	mu.RLock()
	defer mu.RUnlock()

	if inst == nil {
		return
	}

	inst.Incr(ctx, metric)
}

func Decr(ctx context.Context, metric string) {
	mu.RLock()
	defer mu.RUnlock()

	if inst == nil {
		return
	}

	inst.Decr(ctx, metric)
}
