package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"jaeger-logzio/store"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Warn,
		Name:       "jaeger-logzio",
		JSONFormat: true,
	})

	var logzioStore shared.StoragePlugin
	logger.Warn("*************************************************************creating logz storage")
	logzioStore = store.NewLogzioStore("oCwtQDtWjDOMcHXHGGNrnRgkEMxCDuiO","c9b842c7-8527-486f-82de-5bbd8fcb805a", logger)
	logger.Error("**********************************************************starting grpc server")
	grpc.Serve(logzioStore)
}