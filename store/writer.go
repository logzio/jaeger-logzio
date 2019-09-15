package store

import (
	"errors"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/logzio/logzio-go"
	"jaeger-logzio/store/Utils"
	"strings"
)

const (
 HttpsPrefix           = "https://"
 PortSuffix            = ":8071"
 DefaultListenerHost   = "listener.logz.io"
 DropLogsDiskThreshold = 98
)

type loggerWriter struct {
	logger   hclog.Logger
}

//this is to convert between jaeger log messages and logzioSender log messages
func (writer *loggerWriter) Write(msgBytes []byte) (n int, err error) {
	msgString := string(msgBytes)
	if strings.Contains(msgString, "Error") {
		writer.logger.Error(msgString)
	} else {
		writer.logger.Info(msgString)
	}
	return len(msgBytes), nil
}


type logzioSpanWriter struct {
	accountToken	string
	logger   hclog.Logger
	sender   *logzio.LogzioSender
}

func (spanWriter *logzioSpanWriter) WriteSpan(span *model.Span) error {
	spanBytes, err := Utils.TransformToLogzioSpan(span)
	if err != nil {
		return err
	}
	err = spanWriter.sender.Send(spanBytes)
	return err
}

func NewLogzioSpanWriter(config LogzioConfig, url string, logger hclog.Logger) (*logzioSpanWriter, error) {
	if config.Account_Token == "" {
		return nil, errors.New("account token is empty, can't create span writer")
	}
	if url == "" {
		url = DefaultListenerHost
	}
	sender, err := logzio.New(
		config.Account_Token,
		logzio.SetUrl(HttpsPrefix+ url +PortSuffix),
		logzio.SetDebug(&loggerWriter {logger: logger}),
		logzio.SetDrainDiskThreshold(DropLogsDiskThreshold))

	if err != nil {
		return nil, err
	}
	spanWriter := &logzioSpanWriter{
		accountToken:  config.Account_Token,
		logger: logger,
		sender: sender,
	}
	return spanWriter, err
}