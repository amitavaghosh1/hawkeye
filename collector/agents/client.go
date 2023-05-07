package agents

import (
	"context"
	"fmt"
	"hawkeye/protocols"
	"hawkeye/quiver"
	"hawkeye/utils"
	"log"
	"sync"
	"time"
)

const CacheKey = "metrics::%s"

type MetricCollector struct {
	repository quiver.Repository
	metricChan chan protocols.Metric
}

var (
	once      sync.Once
	collector *MetricCollector
)

func NewMetricCollector(repo quiver.Repository) *MetricCollector {
	once.Do(func() {
		mc := &MetricCollector{
			repository: repo,
			metricChan: make(chan protocols.Metric, 1000),
		}

		collector = mc
		go mc.Process(context.Background())
	})

	return collector
}

func (mc MetricCollector) Send(ctx context.Context, metric protocols.Metric) {
	metricStr := fmt.Sprintf("%s:%.1f|%s", metric.Name, metric.Value, metric.MetricType())

	select {
	case mc.metricChan <- metric:
		log.Println("metric sent ", metricStr)
	case <-time.After(100 * time.Millisecond):
		log.Println("metric dropped ", metricStr)
	}
}

// Flushes data every 200 milliseconds or if the batch size is exceeding length
func (mc MetricCollector) Process(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	batchSize := 300
	batch := make([]protocols.Metric, batchSize)
	for {
		select {
		case _metric, ok := <-mc.metricChan:
			if !ok {
				log.Println("collector shut down")
				return
			}

			if len(batch) >= batchSize {
				mc.ProcessBatch(ctx, batch...)
				batch = batch[:0]
			}

			batch = append(batch, _metric)

		case <-ticker.C:
			if len(batch) > 0 {
				mc.ProcessBatch(ctx, batch...)
				batch = batch[:0]
			}
		}

	}
}

func (mc MetricCollector) ProcessBatch(ctx context.Context, metrics ...protocols.Metric) {
	// errchans := make(chan error, len(metrics)/2)

	var wg sync.WaitGroup
	wg.Add(len(metrics))

	for _, metric := range metrics {
		go func(m protocols.Metric) {
			defer wg.Done()

			var err error
			if m.Type == protocols.MetricTypeCounter {
				err = mc.repository.SetCount(ctx, m.Name, utils.Now().UnixMicro(), m.Value)
			}

			if err != nil {
				log.Println("failed to insert metric ", m.Name)
			}
		}(metric)
	}

	wg.Wait()
}
