package store

import (
	"encoding/json"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
)

const (
	levelFatal          = "fatal"
	levelError          = "error"
	valueSuffix         = ".value"
	typeSuffix          = ".type"
	serviceNameProperty = "serviceName"
	tagsProperty        = "tags"
	spanLogType         = "jaegerSpan"
)

type logzioSpan struct {
	TraceID         dbmodel.TraceID        `json:"traceID"`
	OperationName   string                 `json:"operationName,omitempty"`
	SpanID          dbmodel.SpanID         `json:"spanID"`
	References      []dbmodel.Reference    `json:"references"`
	Flags           uint32                 `json:"flags,omitempty"`
	StartTime       uint64                 `json:"startTime"`
	StartTimeMillis uint64                 `json:"startTimeMillis"`
	Timestamp       uint64                 `json:"@timestamp"`
	Duration        uint64                 `json:"duration"`
	Tags            []dbmodel.KeyValue     `json:"JaegerTags,omitempty"`
	Tag             map[string]interface{} `json:"JaegerTag,omitempty"`
	Logs            []dbmodel.Log          `json:"logs"`
	Process         dbmodel.Process        `json:"process,omitempty"`
	Type            string                 `json:"type"`
	//Errors          int                    `json:"errors,omitempty"`
	//Fatals          int                    `json:"fatals,omitempty"`
}

func getTagsValues(tags []model.KeyValue) []string {
	var values []string
	for i := range tags {
		values = append(values, tags[i].VStr)
	}
	return values
}

// TransformToLogzioSpanBytes receives Jaeger span, converts it logzio span and return it as byte array.
// The main differences between Jaeger span and logzio span are arrays which are represented as maps
func TransformToLogzioSpanBytes(span *model.Span) ([]byte, error) {
	spanConverter := dbmodel.NewFromDomain(true, getTagsValues(span.Tags), "@")
	jsonSpan := spanConverter.FromDomainEmbedProcess(span)
	spanProcess := make(map[string]interface{})
	spanProcess[serviceNameProperty] = jsonSpan.Process.ServiceName
	spanProcess[tagsProperty] = transformToLogzioTags(span.Process.Tags)
	logzioSpan := logzioSpan{
		TraceID:         jsonSpan.TraceID,
		OperationName:   jsonSpan.OperationName,
		SpanID:          jsonSpan.SpanID,
		References:      jsonSpan.References,
		Flags:           jsonSpan.Flags,
		StartTime:       jsonSpan.StartTime,
		StartTimeMillis: jsonSpan.StartTimeMillis,
		Timestamp:       jsonSpan.StartTimeMillis,
		Duration:        jsonSpan.Duration,
		Tags:            jsonSpan.Tags,
		Tag:             jsonSpan.Tag,
		Process:         jsonSpan.Process,
		Type:            spanLogType,
		//Errors:          getLogLevelCount(span.Logs, levelError),
		//Fatals:          getLogLevelCount(span.Logs, levelFatal),
	}
	return json.Marshal(logzioSpan)
}

func getLogLevelCount(logs []model.Log, level string) int {
	levelCount := 0
	for _, log := range logs {
		for _, field := range log.Fields {
			if field.Key == "level" && field.Value() == level {
				levelCount++
			}
		}
	}
	return levelCount
}

func transformToLogzioTags(tags []model.KeyValue) map[string]interface{} {
	logzioTags := make(map[string]interface{})
	for _, tag := range tags {
		logzioTags[tag.Key+valueSuffix] = tag.Value()
		if tag.GetVType() != model.ValueType_STRING {
			logzioTags[tag.Key+typeSuffix] = tag.GetVType().String()
		}
	}
	return logzioTags
}
