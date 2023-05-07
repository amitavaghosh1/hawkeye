package raider

import (
	"context"
	"errors"
	"hawkeye/collector/agents"
	"hawkeye/database"
	"hawkeye/protocols"
	"hawkeye/quiver"
	"hawkeye/utils"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"github.com/go-redis/redis/v8"
)

// Have a client server which listens to a unix port
// On receiving a message
// Parse and validate the metric, it has to be a valid datagram protocol
// Enqueue it to a channel

// The Message Controller will be initialized with a channel/queue
// SendCounter(ctx context.Context, metric CounterMetricParams)
// SendGauge(ctx context.Context, metric GaugeMetricParams)
// These both enqueue or drop messages to channel

type MetricServer struct {
	Protocol   string
	Socketfile string
	RedisHost  string
	listener   net.Listener
}

func NewMetricServer(redisHost string) MetricServer {
	Cleanup()

	socket, err := net.Listen(utils.UnixProtocol, utils.SocketFile)
	if err != nil {
		log.Fatal("failed to connect to uds ", err)
	}

	return MetricServer{
		Protocol:   utils.UnixProtocol,
		Socketfile: utils.SocketFile,
		listener:   socket,
		RedisHost:  redisHost,
	}
}

func (m MetricServer) Start(ctx context.Context, closing, done chan struct{}) {
	err := RegisterHandler(database.NewRedisClient(m.RedisHost))
	if err != nil {
		log.Fatal(err)
	}

	defer Cleanup()
	listener := m.listener

	for {
		select {
		case <-closing:
			log.Println("closing listener")
			listener.Close()
			done <- struct{}{}
			return
		case <-ctx.Done():
			log.Println("closing listener because context completed")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			go jsonrpc.ServeConn(conn)
		}
	}
}

func Cleanup() {
	_, err := os.Stat(utils.SocketFile)
	if err == nil {
		os.Remove(utils.SocketFile)
	}
}

func RegisterHandler(client *redis.Client) error {
	handler := &Metric{
		collector: agents.NewMetricCollector(quiver.NewRedisRepo(client)),
	}
	rpc.Register(handler)

	return nil
}

type Metric struct {
	Text      string
	collector *agents.MetricCollector
}

var ErrFailedMetricPush = errors.New("failed to process metrics")

func (m *Metric) Handle(args *Metric, reply *int) error {
	if args == nil {
		*reply = 1
		return nil
	}

	ctx := context.Background()

	metric, err := protocols.ParseDatagram(ctx, args.Text)
	if err != nil {
		log.Println("invalid metric", args.Text)
		*reply = 1
		return ErrFailedMetricPush
	}

	*reply = 0
	// log.Println("sending metric", args.Text)
	m.collector.Send(ctx, metric)
	return nil
}
