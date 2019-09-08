package main

import (
	"flag"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"jaeger-logzio/store"
	"strconv"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Warn,
		Name:       "jaeger-logzio",
		JSONFormat: true,
	})

	var logzioStore shared.StoragePlugin
	var configPath string
	logzioConfig := store.LogzioConfig{}

	flag.StringVar(&configPath, "config", "", "The absolute path to the logz.io plugin's configuration file")
	flag.Parse()
	logger.Error("**************************************************************** config: ", configPath)

	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Error(err.Error())
	} else {
		err = yaml.Unmarshal(yamlFile, &logzioConfig)
		if err != nil {
			logger.Error(err.Error())
		}
	}

	logger.Error("configezzz: ", logzioConfig.Listener_Host, logzioConfig.Listener_Host, strconv.Itoa(len(yamlFile)))

	logger.Warn("*************************************************************creating logz storage")
	logzioStore = store.NewLogzioStore(logzioConfig, logger)
	logger.Error("**********************************************************starting grpc server")
	grpc.Serve(logzioStore)
}