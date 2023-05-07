package main

import (
	"context"
	"hawkeye/collector/agents"
	"hawkeye/collector/aggregator"
	"hawkeye/collector/raider"
	"hawkeye/config"
	"hawkeye/instruments"
	"hawkeye/notifiers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func StartCollector(cfg config.AppConfig, done chan struct{}) {
	log.Print("starting collector")
	closing := make(chan struct{}, 1)

	ctx := context.Background()
	server := raider.NewMetricServer(cfg.RedisHost)
	go server.Start(ctx, closing, done)

	go func() {
		for {
			select {
			case <-done:
				log.Println("requested stop")
				closing <- struct{}{}
				return
			default:
			}
		}
	}()
}

func StartMonitor(cfg config.AppConfig, done chan struct{}) {
	log.Println("starting monitor")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitors := aggregator.ReadMonitoringConfig(cfg.MonitorConfigFile, cfg.ServiceName)

	agent := agents.NewRedisMonitoringAgent(cfg, notifiers.MockMailingService{})
	go agent.Start(ctx, monitors...)

	for {
		select {
		case <-done:
			log.Println("asked to stop")
			return
		default:
		}
	}
}

func main() {
	cfg := config.ReadConfig()

	cc := make(chan struct{}, 1)
	StartCollector(cfg, cc)

	cm := make(chan struct{}, 1)
	go StartMonitor(cfg, cm)

	instruments.InstrumentWithConfig(cfg)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// these normally go in middleware, where you get c.Response.Status
	r.GET("/400", func(c *gin.Context) {
		instruments.Incr(c.Request.Context(), "http.response.400")

		c.JSON(http.StatusBadRequest, gin.H{
			"status": 400,
		})
	})

	r.GET("/500", func(c *gin.Context) {
		instruments.Incr(c.Request.Context(), "http.response.500")

		c.JSON(http.StatusInternalServerError, gin.H{
			"status": 500,
		})
	})

	log.Println("starting server ")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	ctx := context.Background()

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Println("closing monitors and collectors")

	cc <- struct{}{}
	cm <- struct{}{}
	srv.Shutdown(ctx)

}
