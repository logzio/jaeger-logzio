package main

import (
	"flag"
	"jaeger-logzio/store"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
)

const loggerName = "jaeger-logzio"

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       loggerName,
		JSONFormat: true,
	})

	var configPath string
	flag.StringVar(&configPath, "config", "", "The absolute path to the logz.io plugin's configuration file")
	flag.Parse()

	logzioConfig, err := store.ParseConfig(configPath)
	if err != nil {
		logger.Error("can't parse config: ", err.Error())
		panic(err)
	}
	err = logzioConfig.Validate()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	logzioStore := store.NewLogzioStore(logzioConfig, logger)
	grpc.Serve(logzioStore)
	logzioStore.Close()
}
