package main

import (
	"flag"
	"jaeger-logzio/store"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/spf13/viper"
)

const (
	loggerName         = "jaeger-logzio"
	defaultListenerURL = "https://listener.logz.io:8071"
	accountTokenParam  = "accountToken"
	apiTokenParam      = "apiToken"
	listenerURLParam   = "listenerURL"
	emptyString        = ""
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       loggerName,
		JSONFormat: true,
	})
	logger.Error("Initializing logz.io storage")
	var configPath string
	flag.StringVar(&configPath, "config", "", "The absolute path to the logz.io plugin's configuration file")
	flag.Parse()

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault(listenerURLParam, defaultListenerURL)
	v.SetDefault(apiTokenParam, emptyString)

	var logzioConfig store.LogzioConfig
	var err error
	if configPath != "" {
		logzioConfig, err = store.ParseConfig(configPath)
		if err != nil {
			logger.Error("can't parse config: ", err.Error())
			os.Exit(0)
		}
	} else {
		logzioConfig = store.LogzioConfig{
			ListenerURL:  v.GetString(listenerURLParam),
			AccountToken: v.GetString(accountTokenParam),
			APIToken:     v.GetString(apiTokenParam),
		}
	}

	err = logzioConfig.Validate()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(0)
	}

	logzioStore := store.NewLogzioStore(logzioConfig, logger)
	grpc.Serve(logzioStore)
	logzioStore.Close()
}
