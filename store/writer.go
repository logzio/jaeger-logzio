package store

import (
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/logzio/logzio-go"
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
const VALUE_SUFFIX = ".value"
const TYPE_SUFFIX = ".type"
const HTTPS_PREFIX = "https://"
const PORT_SUFFIX = ":8071"

type logzioSpanWriter struct {
	accountToken string
	logger   hclog.Logger
	sender   *logzio.LogzioSender
	spanConverter    dbmodel.FromDomain
}

type loggerWriter struct {
	logger   hclog.Logger
}

func (writer *loggerWriter) Write(p []byte) (n int, err error) {
	writer.logger.Error(string(p))
	return len(p), nil
}

func (spanWriter *logzioSpanWriter) WriteSpan(span *model.Span) error {

	err, spanBytes := spanWriter.transformToLogzioSpan(span)

	if err != nil {
		spanWriter.logger.Warn("************************************************************************************", err.Error())
	}

	err = spanWriter.sender.Send(spanBytes)

	if err != nil {
		spanWriter.logger.Warn("************************************************************************************", err.Error())
	}
	spanWriter.sender.Drain()
	return err
}

func (spanWriter *logzioSpanWriter) transformToLogzioSpan(span *model.Span) (error, []byte) {

	spanString := spanWriter.spanConverter.FromDomainEmbedProcess(span)
	spanBytes, err := json.Marshal(spanString)
	if err != nil {
		spanWriter.logger.Error(err.Error())
	}
	spanMap := make(map[string]interface{})
	err = json.Unmarshal(spanBytes,&spanMap)
	if err != nil {
		spanWriter.logger.Error(err.Error())
	}

	delete(spanMap, JSON_TAGS)
	delete(spanMap, JSON_PROCESS)
	spanMap[JSON_TYPE] = JAEGER_SPAN
	spanMap[JSON_JAEGER_TAGS] = spanWriter.transformToLogzioTags(span.Tags)
	spanMap[JSON_PROCESS_TAGS] = spanWriter.transformToLogzioTags(span.Process.Tags)
	spanMap[JSON_PROCESS_SERVICE_NAME] = span.Process.ServiceName
	spanMap[JSON_TIMESTAMP] = spanMap[JSON_START_TIME_MILLIS]
	spanBytes, err = json.Marshal(spanMap)
	return err, spanBytes
}

func (spanWriter *logzioSpanWriter) transformToLogzioTags(tags []model.KeyValue) map[string]interface{} {
	result := make(map[string]interface{})
	for _, tag := range tags {

		result[tag.Key + VALUE_SUFFIX] = tag.Value()
		if tag.GetVType() != model.ValueType_STRING {
			result[tag.Key + TYPE_SUFFIX] = tag.GetVType().String()
		}
	}
	return result
}

func NewLogzioSpanWriter(accountToken string, url string, logger hclog.Logger) *logzioSpanWriter {
	sender, err := logzio.New(
		accountToken,
		logzio.SetUrl(HTTPS_PREFIX + url + PORT_SUFFIX),
		logzio.SetDebug(&loggerWriter {logger: logger}),
		logzio.SetDrainDiskThreshold(98))

	if err != nil {
		logger.Warn(err.Error(), "********************************************************************")
	}
	spanWriter := &logzioSpanWriter{
		accountToken:  accountToken,
		logger: logger,
		sender: sender,
	}
	return spanWriter
}