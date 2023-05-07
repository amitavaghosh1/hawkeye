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

type MonitoringAgent struct {
	cfg    config.AppConfig
	mailer notifiers.MailingService
	repo   quiver.Repository
}

func NewRedisMonitoringAgent(cfg config.AppConfig, mailer notifiers.MailingService) MonitoringAgent {
	repo := quiver.NewRedisRepo(database.NewRedisClient(cfg.RedisHost))
	return MonitoringAgent{mailer: mailer, cfg: cfg, repo: repo}
}

func (ma MonitoringAgent) Start(ctx context.Context, monitors ...aggregator.Monitor) {
	dones := []chan error{}

	for _, monitor := range monitors {
		if protocols.Is(monitor.Type, protocols.MetricTypeCounter) {
			log.Println("setting up metric counter monitor for ", monitor.Metric)

			done, err := ma.MonitorCounter(ctx, monitor)
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

func (ma MonitoringAgent) MonitorCounter(ctx context.Context, monitor aggregator.Monitor) (chan error, error) {
	cfg := ma.cfg

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
			notifierCfg := notifiers.NotifierConfig{
				ServiceName: cfg.ServiceName,
				Environment: cfg.Environment,
			}

			notifier := notifiers.NewEmailNotifier(
				ma.mailer,
				t,
				notifierCfg,
			)

			opts := []monitors.CounterMonitorOpts{
				monitors.WithThreshold(t.Threshold),
				monitors.WithInterval(interval),
				monitors.WithNotifier(notifier),
				monitors.WithAggregateFunc(aggregator.NewCountAggregator(ma.repo)),
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
