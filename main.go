package main

import (
	"flag"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"jaeger-logzio/store"
)

const LoggerName = "jaeger-logzio"

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       LoggerName,
		JSONFormat: true,
	})

	var configPath string
	logzioConfig := store.LogzioConfig{
		Listener_Host: "https://listener.logz.io:8071",
	}

	flag.StringVar(&configPath, "config", "", "The absolute path to the logz.io plugin's configuration file")
	flag.Parse()
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Error(err.Error())
		panic(err.Error())
	} else {
		err = yaml.Unmarshal(yamlFile, &logzioConfig)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	logzioStore := store.NewLogzioStore(logzioConfig, logger)
	grpc.Serve(logzioStore)
}