package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/opentracing/opentracing-go"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/olivere/elastic"
	"github.com/pkg/errors"
)

// TraceFinder object builds search request from traceIDs and parse the result to traces
type TraceFinder struct {
	logger        hclog.Logger
	sourceFn      sourceFn
	spanConverter dbmodel.ToDomain
	reader        *LogzioSpanReader
}

// NewTraceFinder creates trace finder object
func NewTraceFinder(reader *LogzioSpanReader) TraceFinder {
	return TraceFinder{
		logger:   reader.logger,
		sourceFn: getSourceFn(),
		reader:   reader,
		spanConverter:	dbmodel.NewToDomain("@"),
	}
}

func (finder *TraceFinder) traceIDsMultiSearchRequestBody(traceIDs []model.TraceID, nextTime uint64, endTime time.Time, searchAfterTime map[model.TraceID]uint64) string {
	multiSearchBody := ""
	for _, traceID := range traceIDs {
		finder.logger.Debug(fmt.Sprintf("processing trace %s", traceID.String()))
		traceIDTerm := elastic.NewTermQuery(traceIDField, traceID.String())
		rangeQuery := elastic.NewRangeQuery(startTimeField).Gte(nextTime).Lte(model.TimeAsEpochMicroseconds(endTime))
		query := elastic.NewBoolQuery().Filter(traceIDTerm, rangeQuery)
		if val, ok := searchAfterTime[traceID]; ok {
			nextTime = val
		}
		source := finder.sourceFn(query, nextTime)
		searchRequest := elastic.NewSearchRequest().
			IgnoreUnavailable(true).
			Source(source)
		requestBody, err := searchRequest.Body()
		if err != nil {
			finder.logger.Warn(fmt.Sprintf("can't create search request for traceID %s, skipping..", traceID.String()))
			continue
		}
		finder.logger.Debug(fmt.Sprintf("creating logzio search request: %s", requestBody))
		// add search {}\n to prefix and \n to suffix of the search request to match it to multiSearch format
		multiSearchBody = fmt.Sprintf("%s{}\n%s\n", multiSearchBody, requestBody)
	}
	return multiSearchBody
}

func (finder *TraceFinder) getTracesMap(traceIDs []model.TraceID, startTime time.Time, endTime time.Time) map[model.TraceID]*model.Trace {
	nextTime := model.TimeAsEpochMicroseconds(startTime)
	searchAfterTime := make(map[model.TraceID]uint64)
	totalDocumentsFetched := make(map[model.TraceID]int)
	tracesMap := make(map[model.TraceID]*model.Trace)
	for {
		finder.logger.Debug("TraceIds left to process for multiRead: " + string(len(traceIDs)))
		if len(traceIDs) == 0 {
			break
		}
		multiSearchBody := finder.traceIDsMultiSearchRequestBody(traceIDs, nextTime, endTime, searchAfterTime)
		// set traceIDs to empty
		traceIDs = nil

		finder.logger.Error(multiSearchBody)
		results, err := finder.reader.getMultiSearchResult(multiSearchBody)
		if err != nil || results.Responses == nil || len(results.Responses) == 0 {
			if err != nil {
				finder.logger.Error(err.Error())
			}
			break
		}

		for _, result := range results.Responses {
			if result.Hits == nil || len(result.Hits.Hits) == 0 {
				continue
			}
			spans, err := finder.collectSpans(result.Hits.Hits)
			if err != nil {
				finder.logger.Warn("can't collect spans form result")
				continue
			}
			lastSpan := spans[len(spans)-1]

			if traceSpan, ok := tracesMap[lastSpan.TraceID]; ok {
				traceSpan.Spans = append(traceSpan.Spans, spans...)
			} else {
				tracesMap[lastSpan.TraceID] = &model.Trace{Spans: spans}
			}

			totalDocumentsFetched[lastSpan.TraceID] = totalDocumentsFetched[lastSpan.TraceID] + len(result.Hits.Hits)
			if totalDocumentsFetched[lastSpan.TraceID] < int(result.TotalHits()) {
				traceIDs = append(traceIDs, lastSpan.TraceID)
				searchAfterTime[lastSpan.TraceID] = model.TimeAsEpochMicroseconds(lastSpan.StartTime)
			}
		}
	}
	return tracesMap
}

func (finder *TraceFinder) collectSpans(esSpansRaw []*elastic.SearchHit) ([]*model.Span, error) {
	spans := make([]*model.Span, len(esSpansRaw))

	for i, esSpanRaw := range esSpansRaw {
		jsonSpan, err := unmarshalJSONSpan(esSpanRaw)
		if err != nil {
			return nil, errors.Wrap(err, "Marshalling JSON to span object failed")
		}
		dbSpan := jsonSpan.TransformToDbModelSpan()
		span, err := finder.spanConverter.SpanToDomain(dbSpan)
		if err != nil {
			return nil, errors.Wrap(err, "Converting JSONSpan to domain Span failed")
		}
		spans[i] = span
	}
	return spans, nil
}

func (finder *TraceFinder) multiRead(ctx context.Context, traceIDs []model.TraceID, startTime, endTime time.Time) ([]*model.Trace, error) {
	if len(traceIDs) == 0 {
		return []*model.Trace{}, nil
	}
	tracesMap := finder.getTracesMap(traceIDs, startTime, endTime)

	var traces []*model.Trace
	for _, trace := range tracesMap {
		traces = append(traces, trace)
	}
	return traces, nil
}

func (finder *TraceFinder) findTraceIDsStrings(ctx context.Context, traceQuery *spanstore.TraceQueryParameters) ([]string, error) {
	childSpan, _ := opentracing.StartSpanFromContext(ctx, "findTraceIDsStrings")
	defer childSpan.Finish()

	aggregation := buildTraceIDAggregation(traceQuery.NumTraces)
	boolQuery := finder.buildFindTraceIDsQuery(traceQuery)

	searchService := elastic.NewSearchRequest().
		Size(0). // set to 0 because we don't want actual documents.
		Aggregation(traceIDAggregation, aggregation).
		IgnoreUnavailable(true).
		Query(boolQuery)

	requestBody, err := searchService.Body()
	if err != nil {
		finder.logger.Warn("can't create search request for trace query")
		return nil, err
	}
	requestBody = fmt.Sprintf("{}\n%s\n", requestBody)
	finder.logger.Error(string(requestBody))
	multiSearchResult, err := finder.reader.getMultiSearchResult(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "Search service failed")
	}
	searchResult := multiSearchResult.Responses[0]
	bucket, found := searchResult.Aggregations.Terms(traceIDAggregation)
	if !found {
		return nil, ErrUnableToFindTraceIDAggregation
	}

	traceIDBuckets := bucket.Buckets
	return bucketToStringArray(traceIDBuckets)
}

func (finder *TraceFinder) buildTagQuery(k string, v string) elastic.Query {
	objectTagListLen := len(objectTagFieldList)
	queries := make([]elastic.Query, len(nestedTagFieldList)+objectTagListLen)
	kd := finder.spanConverter.ReplaceDot(k)
	for i := range objectTagFieldList {
		queries[i] = buildObjectQuery(objectTagFieldList[i], kd, v)
	}
	for i := range nestedTagFieldList {
		queries[i+objectTagListLen] = buildNestedQuery(nestedTagFieldList[i], k, v)
	}

	// but configuration can change over time
	return elastic.NewBoolQuery().Should(queries...)
}

func (finder *TraceFinder) buildFindTraceIDsQuery(traceQuery *spanstore.TraceQueryParameters) elastic.Query {
	boolQuery := elastic.NewBoolQuery()

	//add duration query
	if traceQuery.DurationMax != 0 || traceQuery.DurationMin != 0 {
		durationQuery := buildDurationQuery(traceQuery.DurationMin, traceQuery.DurationMax)
		boolQuery.Must(durationQuery)
	}

	//add startTime query
	startTimeQuery := buildStartTimeQuery(traceQuery.StartTimeMin, traceQuery.StartTimeMax)
	boolQuery.Must(startTimeQuery)

	//add process.serviceName query
	if traceQuery.ServiceName != "" {
		serviceNameQuery := buildServiceNameQuery(traceQuery.ServiceName)
		boolQuery.Must(serviceNameQuery)
	}

	//add operationName query
	if traceQuery.OperationName != "" {
		operationNameQuery := buildOperationNameQuery(traceQuery.OperationName)
		boolQuery.Must(operationNameQuery)
	}

	for k, v := range traceQuery.Tags {
		tagQuery := finder.buildTagQuery(k, v)
		boolQuery.Must(tagQuery)
	}
	return boolQuery
}
