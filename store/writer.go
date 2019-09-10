package store

import (
	"encoding/json"
	"errors"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/logzio/logzio-go"
	"strings"
)

const JSON_TYPE = "type"
const JAEGER_SPAN = "JaegerSpan"
const JSON_JAEGER_TAGS = "JaegerTags"
const JSON_TAGS = "tags"
const JSON_PROCESS = "process"
const JSON_PROCESS_TAGS = "process.tags"
const JSON_PROCESS_SERVICE_NAME = "process.serviceName"
const JSON_TIMESTAMP = "@timestamp"
const JSON_START_TIME_MILLIS = "startTimeMillis"
const JSON_WARNINGS = "warnings"
const JSON_ERRORS = "errors"
const LEVEL_WARNING = "warning"
const LEVEL_ERROR = "error"
const VALUE_SUFFIX = ".value"
const TYPE_SUFFIX = ".type"
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
	spanBytes, err := transformToLogzioSpan(span)
	if err != nil {
		return err
	}
	err = spanWriter.sender.Send(spanBytes)
	return err
}

func transformToLogzioSpan(span *model.Span) ([]byte, error) {
	spanConverter := dbmodel.FromDomain{}
	spanString := spanConverter.FromDomainEmbedProcess(span)
	spanBytes, err := json.Marshal(spanString)
	if err != nil {
		return nil, err
	}
	spanMap := make(map[string]interface{})
	err = json.Unmarshal(spanBytes,&spanMap)
	if err != nil {
		return nil, err
	}

	delete(spanMap, JSON_TAGS)
	delete(spanMap, JSON_PROCESS)
	spanMap[JSON_TYPE] = JAEGER_SPAN
	spanMap[JSON_JAEGER_TAGS] = transformToLogzioTags(span.Tags)
	spanMap[JSON_PROCESS_TAGS] = transformToLogzioTags(span.Process.Tags)
	spanMap[JSON_PROCESS_SERVICE_NAME] = span.Process.ServiceName
	spanMap[JSON_TIMESTAMP] = spanMap[JSON_START_TIME_MILLIS]
	spanMap[JSON_WARNINGS] = getLogLevelCount(span.Logs, LEVEL_WARNING)
	spanMap[JSON_ERRORS] = getLogLevelCount(span.Logs, LEVEL_ERROR)
	return json.Marshal(spanMap)
}

func getLogLevelCount(logs []model.Log, level string) int {
	levelCount := 0
	for _,log := range logs {
		for _, field := range log.Fields {
			if field.Key == "level" && field.Value() == level {
				levelCount++
			}
		}
	}
	return levelCount
}

func transformToLogzioTags(tags []model.KeyValue) map[string]interface{} {
	result := make(map[string]interface{})
	for _, tag := range tags {
		result[tag.Key + VALUE_SUFFIX] = tag.Value()
		if tag.GetVType() != model.ValueType_STRING {
			result[tag.Key + TYPE_SUFFIX] = tag.GetVType().String()
		}
	}
	return result
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