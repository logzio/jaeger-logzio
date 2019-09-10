package store

import (
	"errors"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/logzio/logzio-go"
	"jaeger-logzio/store/Utils"
	"strings"
)

const HTTPS_PREFIX = "https://"
const PORT_SUFFIX = ":8071"
const DEFAULT_LISTENER_HOST = "listener.logz.io"
const DROP_LOGS_DISK_THRESHOLD = 98

type logzioSpanWriter struct {
	accountToken	string
	logger   hclog.Logger
	sender   *logzio.LogzioSender
}

type loggerWriter struct {
	logger   hclog.Logger
}

func (writer *loggerWriter) Write(msgBytes []byte) (n int, err error) {
	msgString := string(msgBytes)
	if strings.Contains(msgString, "Error") {
		writer.logger.Error(msgString)
	} else {
		writer.logger.Info(msgString)
	}
	return len(msgBytes), nil
}

func (spanWriter *logzioSpanWriter) WriteSpan(span *model.Span) error {
	spanBytes, err := Utils.TransformToLogzioSpan(span)
	if err != nil {
		return err
	}
	err = spanWriter.sender.Send(spanBytes)
	return err
}

func NewLogzioSpanWriter(accountToken string, url string, logger hclog.Logger) (*logzioSpanWriter, error) {
	if accountToken == "" {
		return nil, errors.New("account token is empty, can't create span writer")
	}
	if url == "" {
		url = DEFAULT_LISTENER_HOST
	}
	sender, err := logzio.New(
		accountToken,
		logzio.SetUrl(HTTPS_PREFIX + url + PORT_SUFFIX),
		logzio.SetDebug(&loggerWriter {logger: logger}),
		logzio.SetDrainDiskThreshold(DROP_LOGS_DISK_THRESHOLD))

	if err != nil {
		return nil, err
	}
	spanWriter := &logzioSpanWriter{
		accountToken:  accountToken,
		logger: logger,
		sender: sender,
	}
	return spanWriter, err
}