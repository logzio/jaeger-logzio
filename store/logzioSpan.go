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
	jaegerType			= "jaegerSpan"
)

type logzioSpan struct {
	TraceID         dbmodel.TraceID        `json:"traceId"`
	OperationName   string                 `json:"operationName,omitempty"`
	SpanID          dbmodel.SpanID         `json:"spanId"`
	References      []dbmodel.Reference    `json:"references,omitempty"` //References can be empty when the span is a root
	Flags           uint32                 `json:"flags"`
	StartTime       uint64                 `json:"startTime"`
	StartTimeMillis uint64                 `json:"startTimeMillis"`
	Timestamp       uint64                 `json:"@timestamp"`
	Duration        uint64                 `json:"duration"`
	Tags            map[string]interface{} `json:"JaegerTags"`
	Logs            []dbmodel.Log          `json:"logs,omitempty"`
	Process         map[string]interface{} `json:"process,omitempty"`
	Errors          int                    `json:"errors"`
	Fatals          int                    `json:"fatals"`
	Type			string				   `json:"type"`
	Warnings		[]string			   `json:"warnings"`
}

// TransformToLogzioSpanBytes receives Jaeger span, converts it logzio span and return it as byte array.
// The main differences between Jaeger span and logzio span are arrays which are represented as maps
func TransformToLogzioSpanBytes(span *model.Span) ([]byte, error) {
	// todo - check with Yogev why it's always empty.
	spanWarnings := span.Warnings
	spanConverter := dbmodel.FromDomain{}
	jsonSpan := spanConverter.FromDomainEmbedProcess(span)
	spanProcess := make(map[string]interface{})
	spanProcess[serviceNameProperty] = jsonSpan.Process.ServiceName
	spanProcess[tagsProperty] = transformToLogzioTags(span.Process.Tags)
	errorCount, fatelCount := getLogLevelsCount(span.Logs)
	logzioSpan := logzioSpan {
		TraceID:         jsonSpan.TraceID,
		OperationName:   jsonSpan.OperationName,
		SpanID:          jsonSpan.SpanID,
		References:      jsonSpan.References,
		Flags:           jsonSpan.Flags,
		StartTime:       jsonSpan.StartTime,
		StartTimeMillis: jsonSpan.StartTimeMillis,
		Timestamp:       jsonSpan.StartTimeMillis,
		Duration:        jsonSpan.Duration,
		Tags:            transformToLogzioTags(span.Tags),
		Process:         spanProcess,
		Errors:          errorCount,
		Fatals:          fatelCount,
		Type:			 jaegerType,
		Logs:			 jsonSpan.Logs,
		Warnings:		 spanWarnings,
	}
	return json.Marshal(logzioSpan)
}

func getLogLevelsCount(logs []model.Log) (int, int) {
	errs := 0
	fatels := 0
	for _, log := range logs {
		for _, field := range log.Fields {
			if field.Key == "level" {
				if field.Value() == levelError {
					errs++
				} else if field.Value() == levelFatal {
					fatels++
				}
				// todo - check if we can use break here(saves time) and assume only one log level per log.
			}
		}
	}
	return errs, fatels
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
