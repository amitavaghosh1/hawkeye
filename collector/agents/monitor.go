package agents

import (
	"context"
	"errors"
	"hawkeye/collector/aggregator"
	"hawkeye/collector/monitors"
	"hawkeye/config"
	"hawkeye/database"
	"hawkeye/notifiers"
	"hawkeye/protocols"
	"hawkeye/quiver"
	"hawkeye/utils"
	"log"
	"sync"
	"time"
)

// this will take a list of Monitor and depending on metric type, it will start
// <MetricType>Monitor. Example: CounterMonitor

type AgentConfig struct {
	Repo quiver.Repository
}

func Start(ctx context.Context, cfg config.AppConfig, monitors ...aggregator.Monitor) {
	repo := quiver.NewRedisRepo(database.NewRedisClient(cfg.RedisHost))
	deps := AgentConfig{Repo: repo}

	dones := []chan error{}

	for _, monitor := range monitors {
		if protocols.Is(monitor.Type, protocols.MetricTypeCounter) {
			log.Println("setting up metric counter monitor for ", monitor.Metric)

			done, err := MonitorCounter(ctx, cfg, deps, monitor)
			if err != nil {
				log.Println("could not start monitoring counters")
				continue
			}
			dones = append(dones, done)
		}
	}

	for err := range utils.JoinErrors(dones...) {
		if err != nil {
			log.Println("monitor stopped ", err)
		}
	}

}

const DefaultIntervalInSeconds time.Duration = 60

var ErrMonitoringStopped = errors.New("monitoring_stopped")

func MonitorCounter(ctx context.Context, cfg config.AppConfig, deps AgentConfig, monitor aggregator.Monitor) (chan error, error) {

	if len(monitor.Triggers) == 0 {
		return nil, errors.New("no_triggers_registered")
	}

	log.Println("starting count metric monitor ", monitor.Metric)
	// log.Printf("%+v\n", monitor)

	interval := time.Duration(monitor.IntervalInSeconds)
	if interval == 0 {
		interval = DefaultIntervalInSeconds
	}
	interval = interval * time.Second

	done := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(len(monitor.Triggers))

	for _, trigger := range monitor.Triggers {
		go func(t aggregator.Trigger) {
			mailer := notifiers.MockMailingService{}
			notifierCfg := notifiers.NotifierConfig{
				ServiceName: cfg.ServiceName,
				Environment: cfg.Environment,
			}

			notifier := notifiers.NewEmailNotifier(
				mailer,
				t,
				notifierCfg,
			)

			opts := []monitors.CounterMonitorOpts{
				monitors.WithThreshold(t.Threshold),
				monitors.WithInterval(interval),
				monitors.WithNotifier(notifier),
				monitors.WithAggregateFunc(aggregator.NewCountAggregator(deps.Repo)),
			}

			cm := monitors.NewCounterMonitor(
				monitor.Metric,
				cfg.Environment,
				opts...,
			)

			cm.Start(ctx, &wg)
		}(trigger)
	}

	go func() {
		wg.Wait()
		log.Println("all monitors resolved ")
		done <- ErrMonitoringStopped
		close(done)
	}()

	return done, nil
}
