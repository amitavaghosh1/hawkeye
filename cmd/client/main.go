package main

import (
	"context"
	"hawkeye/collector/raider"
	"hawkeye/config"
	"hawkeye/utils"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.AppConfig{
		RedisHost: "localhost:6379",
	}

	cfg.ValidateConnections()

	log.Println("listening at", utils.SocketFile)
	closing := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	server := raider.NewMetricServer(cfg.RedisHost)
	go server.Start(context.Background(), closing, done)

	select {
	case sig := <-c:
		log.Println("stopping from ", sig.String())
		closing <- struct{}{}
		return
	case <-done:
		log.Println("closed now cleanup")
		return
	}
}
