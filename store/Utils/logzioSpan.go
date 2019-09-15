package Utils

import (
	"encoding/json"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
)

const JsonProcessServiceName = "process.serviceName"
const JsonType = "type"
const JaegerSpan = "JaegerSpan"
const JsonJaegerTags = "JaegerTags"
const JsonTags = "tags"
const JsonProcess = "process"
const JsonProcessTags = "process.tags"
const JsonTimestamp = "@timestamp"
const JsonStartTimeMillis = "startTimeMillis"
const JsonFatals = "fatals"
const JsonErrors = "errors"
const LevelFatal = "fatal"
const LevelError = "error"
const ValueSuffix = ".value"
const TypeSuffix = ".type"

func TransformToLogzioSpan(span *model.Span) ([]byte, error) {
	spanConverter := dbmodel.FromDomain{}
	spanString := spanConverter.FromDomainEmbedProcess(span)
	spanBytes, err := json.Marshal(spanString)
	if err != nil {
		return nil, err
	}
	spanMap := make(map[string]interface{})
	err = json.Unmarshal(spanBytes, &spanMap)
	if err != nil {
		return nil, err
	}

	delete(spanMap, JsonTags)
	delete(spanMap, JsonProcess)
	spanMap[JsonType] = JaegerSpan
	spanMap[JsonJaegerTags] = transformToLogzioTags(span.Tags)
	spanMap[JsonProcessTags] = transformToLogzioTags(span.Process.Tags)
	spanMap[JsonProcessServiceName] = span.Process.ServiceName
	spanMap[JsonTimestamp] = spanMap[JsonStartTimeMillis]
	spanMap[JsonFatals] = getLogLevelCount(span.Logs, LevelFatal)
	spanMap[JsonErrors] = getLogLevelCount(span.Logs, LevelError)
	return json.Marshal(spanMap)
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
			logzioTags[tag.Key + TypeSuffix] = tag.GetVType()
		}
	}
	return logzioTags
}