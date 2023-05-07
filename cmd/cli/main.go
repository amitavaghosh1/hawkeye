package main

import (
	"context"
	"hawkeye/collector/raider"
	"hawkeye/config"
	"hawkeye/instruments"
	"hawkeye/utils"
	"log"
)

func checkErr(err error, msg string) {
	if err == nil {
		return
	}

	log.Fatal(msg, err)

}

func ReadEnvConfig() {
	cfg := config.ReadConfig()
	log.Printf("%+v\n", cfg)
}

func MakeRPCCall(metric string) {
	rpcClient := utils.InitClientUnix()
	var reply int

	err := rpcClient.Call("Metric.Handle", raider.Metric{Text: metric}, &reply)
	checkErr(err, "failed to make rpc call")

	log.Println("metric sent at ", utils.ToUnix(utils.Now()))
}

func SendMetric() {
	for i := 0; i < 10; i++ {
		MakeRPCCall("http.response.400:1|c")
	}

	for i := 0; i < 10; i++ {
		MakeRPCCall("http.response.500:1|c")
	}
}

func SendMetricWithInstrument() {
	instruments.InstrumentWithConfig(config.AppConfig{})

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		instruments.Incr(ctx, "http.response.400")
	}

	// for i := 0; i < 10; i++ {
	// 	MakeRPCCall("http.response.500:1|c")
	// }
}

func main() {
	log.SetFlags(log.Llongfile)
	SendMetricWithInstrument()

}
