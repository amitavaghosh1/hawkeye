package main

import (
	"context"
	"hawkeye/collector/agents"
	"hawkeye/collector/aggregator"
	"hawkeye/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func RunMetricMonitor() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.ReadConfig()
	cfg.ValidateConnections()

	monitors := aggregator.ReadMonitoringConfig(cfg.MonitorConfigFile, cfg.ServiceName)

	go agents.Start(ctx, cfg, monitors...)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-c:
			log.Println("received signal ", sig.String())
			return
		default:
		}
	}
}

func main() {
	log.SetFlags(log.Llongfile)
	RunMetricMonitor()
}
