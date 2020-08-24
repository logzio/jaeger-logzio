package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/logzio/jaeger-logzio/store/objects"
	"time"

	"github.com/jaegertracing/jaeger/storage/spanstore"

	"github.com/jaegertracing/jaeger/model"
	"github.com/olivere/elastic"
	"github.com/pkg/errors"
)

func convertTraceIDsStringsToModels(traceIDs []string) ([]model.TraceID, error) {
	traceIDsModels := make([]model.TraceID, len(traceIDs))
	for i, ID := range traceIDs {
		traceID, err := model.TraceIDFromString(ID)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Making traceID from string '%s' failed", ID))
		}

		traceIDsModels[i] = traceID
	}
	return traceIDsModels, nil
}

func unmarshalJSONSpan(esSpanRaw *elastic.SearchHit) (*objects.LogzioSpan, error) {
	esSpanInByteArray := esSpanRaw.Source

	var jsonSpan objects.LogzioSpan

	decoder := json.NewDecoder(bytes.NewReader(*esSpanInByteArray))
	decoder.UseNumber()
	if err := decoder.Decode(&jsonSpan); err != nil {
		return nil, err
	}
	return &jsonSpan, nil
}

func buildTraceIDAggregation(numOfTraces int) elastic.Aggregation {
	return elastic.NewTermsAggregation().
		Size(numOfTraces).
		Field(traceIDField).
		Order(startTimeField, false).
		SubAggregation(startTimeField, buildTraceIDSubAggregation())
}

func buildTraceIDSubAggregation() elastic.Aggregation {
	return elastic.NewMaxAggregation().
		Field(startTimeField)
}

func buildDurationQuery(durationMin time.Duration, durationMax time.Duration) elastic.Query {
	minDurationMicros := model.DurationAsMicroseconds(durationMin)
	maxDurationMicros := defaultMaxDuration
	if durationMax != 0 {
		maxDurationMicros = model.DurationAsMicroseconds(durationMax)
	}
	return elastic.NewRangeQuery(durationField).Gte(minDurationMicros).Lte(maxDurationMicros)
}

func buildStartTimeQuery(startTimeMin time.Time, startTimeMax time.Time) elastic.Query {
	minStartTimeMicros := model.TimeAsEpochMicroseconds(startTimeMin)
	maxStartTimeMicros := model.TimeAsEpochMicroseconds(startTimeMax)
	return elastic.NewRangeQuery(startTimeField).Gte(minStartTimeMicros).Lte(maxStartTimeMicros)
}

func buildServiceNameQuery(serviceName string) elastic.Query {
	return elastic.NewMatchQuery(serviceNameField, serviceName)
}

func buildOperationNameQuery(operationName string) elastic.Query {
	return elastic.NewMatchQuery(operationNameField, operationName)
}

func buildNestedQuery(field string, k string, v string) elastic.Query {
	keyField := fmt.Sprintf("%s.%s", field, tagKeyField)
	valueField := fmt.Sprintf("%s.%s", field, tagValueField)
	keyQuery := elastic.NewMatchQuery(keyField, k)
	valueQuery := elastic.NewMatchQuery(valueField, v)
	tagBoolQuery := elastic.NewBoolQuery().Must(keyQuery, valueQuery)
	return elastic.NewNestedQuery(field, tagBoolQuery)
}

func buildObjectQuery(field string, k string, v string) elastic.Query {
	keyField := fmt.Sprintf("%s.%s", field, k)
	keyQuery := elastic.NewMatchQuery(keyField, v)
	return elastic.NewBoolQuery().Must(keyQuery)
}

func bucketToStringArray(buckets []*elastic.AggregationBucketKeyItem) ([]string, error) {
	strings := make([]string, len(buckets))
	for i, keyitem := range buckets {
		str, ok := keyitem.Key.(string)
		if !ok {
			return nil, errors.New("Non-string key found in aggregation")
		}
		strings[i] = str
	}
	return strings, nil
}

func validateQuery(p *spanstore.TraceQueryParameters) error {
	if p == nil {
		return ErrMalformedRequestObject
	}
	if p.ServiceName == "" && len(p.Tags) > 0 {
		return ErrServiceNameNotSet
	}
	if p.StartTimeMin.IsZero() || p.StartTimeMax.IsZero() {
		return ErrStartAndEndTimeNotSet
	}
	if p.StartTimeMax.Before(p.StartTimeMin) {
		return ErrStartTimeMinGreaterThanMax
	}
	if p.DurationMin != 0 && p.DurationMax != 0 && p.DurationMin > p.DurationMax {
		return ErrDurationMinGreaterThanMax
	}
	if p.NumTraces > 1000 {
		p.NumTraces = 1000
	}
	return nil
}
