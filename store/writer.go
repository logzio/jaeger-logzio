package store

import (
	"strings"

	"github.com/logzio/logzio-go"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
)

const (
	httpsPrefix           = "https://"
	portSuffix            = ":8071"
	defaultListenerHost   = "listener.logz.io"
	dropLogsDiskThreshold = 98
)

type loggerWriter struct {
	logger hclog.Logger
}

//this is to convert between jaeger log messages and logzioSender log messages
func (writer *loggerWriter) Write(msgBytes []byte) (n int, err error) {
	msgString := string(msgBytes)
	if strings.Contains(strings.ToLower(msgString), "error") {
		writer.logger.Error(msgString)
	} else {
		writer.logger.Info(msgString)
	}
	return len(msgBytes), nil
}

// LogzioSpanWriter is a struct which holds logzio span writer properties
type LogzioSpanWriter struct {
	accountToken string
	logger       hclog.Logger
	sender       *logzio.LogzioSender
}

// WriteSpan receives a Jaeger span, converts it to logzio span and send it to logzio
func (spanWriter *LogzioSpanWriter) WriteSpan(span *model.Span) error {
	spanBytes, err := TransformToLogzioSpan(span)
	if err != nil {
		return err
	}
	err = spanWriter.sender.Send(spanBytes)
	return err
}

// Close stops and drain logzio sender
func (spanWriter *LogzioSpanWriter) Close() {
	spanWriter.sender.Stop()
}

// NewLogzioSpanWriter creates a new logzio span writer for jaeger
func NewLogzioSpanWriter(config LogzioConfig, url string, logger hclog.Logger) (*LogzioSpanWriter, error) {
	if url == "" {
		url = defaultListenerHost
	}
	sender, err := logzio.New(
		config.AccountToken,
		logzio.SetUrl(httpsPrefix+url+portSuffix),
		logzio.SetDebug(&loggerWriter{logger: logger}),
		logzio.SetDrainDiskThreshold(dropLogsDiskThreshold))

	if err != nil {
		return nil, err
	}
	spanWriter := &LogzioSpanWriter{
		accountToken: config.AccountToken,
		logger:       logger,
		sender:       sender,
	}
	return spanWriter, err
}
