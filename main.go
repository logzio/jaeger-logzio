package main

import (
	"flag"
	"jaeger-logzio/store"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
)

const (
	loggerName = "jaeger-logzio"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       loggerName,
		JSONFormat: true,
	})
	logger.Info("Initializing logz.io storage")
	var configPath string
	flag.StringVar(&configPath, "config", "", "The absolute path to the logz.io plugin's configuration file")
	flag.Parse()

	logzioConfig, err := store.ParseConfig(configPath, logger)
	if err != nil {
		logger.Error("can't parse config: ", err.Error())
		os.Exit(0)
	}

	logger.Info(logzioConfig.String())
	logzioStore := store.NewLogzioStore(*logzioConfig, logger)
	grpc.Serve(logzioStore)
	logzioStore.Close()
}
