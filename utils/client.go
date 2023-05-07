package utils

import (
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
)

const (
	UnixProtocol = "unix"
	SocketFile   = "/tmp/hawkeye.sock"
)

type RPCClient interface {
	Call(serviceMethod string, args any, reply any) error
	Close() error
	Go(serviceMethod string, args any, reply any, done chan *rpc.Call) *rpc.Call
}

type MockClient struct{}

func (m *MockClient) Call(serviceMethod string, args any, reply any) error {
	return nil
}

func (m *MockClient) Close() error {
	return nil
}

func (m *MockClient) Go(serviceMethod string, args any, reply any, done chan *rpc.Call) *rpc.Call {
	return nil
}

func InitClientUnix() RPCClient {
	var (
		err       error
		rpcClient *rpc.Client
	)

	rpcClient, err = jsonrpc.Dial(UnixProtocol, SocketFile)
	if err != nil {
		log.Println("failed to connect to socket ", err)
		return &MockClient{}
	}

	return rpcClient
}
