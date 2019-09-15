package store

import (
	"encoding/json"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
)

const (
	LevelFatal = "fatal"
 	LevelError = "error"
 	ValueSuffix = ".value"
 	TypeSuffix = ".type"
	ServiceNameProperty = "serviceName"
	tagsProperty = "tags"
)

type logzioSpan struct {
	TraceID              dbmodel.TraceID       `json:"trace_id"`
	OperationName        string        `json:"operation_name,omitempty"`
	SpanID               dbmodel.SpanID        `json:"span_id"`
	References           []dbmodel.Reference `json:"references"`
	Flags                uint32         `json:"flags"`
	StartTime            uint64     `json:"start_time"`
	StartTimeMillis      uint64     `json:"start_time"`
	Timestamp			 uint64		`json:"@timestamp"`
	Duration             uint64 `json:"duration"`
	Tags                 map[string]interface{}    `json:"JaegerTags"`
	Logs                []dbmodel.Log  `json:"logs"`
	Process              map[string]interface{}      `json:"process,omitempty"`
	Errors              int      `json:"errors,omitempty"`
	Fatals				int 	 `json:"fatals,omitempty"`
}

func TransformToLogzioSpan(span *model.Span) ([]byte, error) {
	spanConverter := dbmodel.FromDomain{}
	jsonSpan := spanConverter.FromDomainEmbedProcess(span)
	spanProcess := make(map[string]interface{})
	spanProcess[ServiceNameProperty] = jsonSpan.Process.ServiceName
	spanProcess[tagsProperty] = transformToLogzioTags(span.Process.Tags)
	logzioSpan := logzioSpan{
		TraceID:		jsonSpan.TraceID,
		OperationName:	jsonSpan.OperationName,
		SpanID:	jsonSpan.SpanID,
		References:	jsonSpan.References,
		Flags:	jsonSpan.Flags,
		StartTime:	jsonSpan.StartTime,
		StartTimeMillis:	jsonSpan.StartTimeMillis,
		Timestamp:	jsonSpan.StartTimeMillis,
		Duration:	jsonSpan.Duration,
		Tags:	transformToLogzioTags(span.Tags),
		Process:	spanProcess,
		Errors:	getLogLevelCount(span.Logs, LevelError),
		Fatals:	getLogLevelCount(span.Logs, LevelFatal),

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
		logzioTags[tag.Key + ValueSuffix] = tag.Value()
		if tag.GetVType() != model.ValueType_STRING {
			logzioTags[tag.Key + TypeSuffix] = tag.GetVType().String()
		}
	}
	return logzioTags
}