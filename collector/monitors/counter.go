package monitors

import (
	"context"
	"fmt"
	"hawkeye/collector/aggregator"
	"hawkeye/notifiers"
	"log"
	"time"

	"sync"
)

type CounterMonitor struct {
	name      string
	threshold float32
	interval  time.Duration
	collector aggregator.Aggregator
	notify    notifiers.Notifier
	env       string
	closing   chan chan struct{}
}

type CounterMonitorOpts func(c *CounterMonitor)

func NewCounterMonitor(name, env string, opts ...CounterMonitorOpts) *CounterMonitor {
	cm := &CounterMonitor{
		name:    name,
		env:     env,
		closing: make(chan chan struct{}),
	}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

func WithThreshold(threshold float32) CounterMonitorOpts {
	return func(c *CounterMonitor) {
		c.threshold = threshold
	}
}

func WithInterval(interval time.Duration) CounterMonitorOpts {
	return func(c *CounterMonitor) {
		c.interval = interval
	}
}

func WithAggregateFunc(fn aggregator.Aggregator) CounterMonitorOpts {
	return func(c *CounterMonitor) {
		c.collector = fn
	}
}

func WithNotifier(n notifiers.Notifier) CounterMonitorOpts {
	return func(c *CounterMonitor) {
		c.notify = n
	}
}

func (c *CounterMonitor) Start(ctx context.Context, w *sync.WaitGroup) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	defer close(c.closing)
	defer w.Done()

	for {
		select {
		case <-ctx.Done():
			log.Println("quiting from context")
			return
		case done := <-c.closing:
			log.Println("closing from stop")
			done <- struct{}{}
			return

		case <-ticker.C:
			// log.Println("collecting ", c.name)
			count := c.collector.Collect(ctx, c.name, c.interval)

			if count >= c.threshold {
				text := fmt.Sprintf("%s has exceeded threshold by %.3f in %s", c.name, count-c.threshold, c.env)
				err := c.notify.Send(ctx, map[string]interface{}{"count": text})
				if err != nil {
					log.Println("failed to notify ", text)
				}
			}
		}
	}
}

func (c *CounterMonitor) Stop() {
	done := make(chan struct{})
	c.closing <- done
	<-done
}
