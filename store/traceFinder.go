package store

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/logzio/jaeger-logzio/store/objects"
	"github.com/opentracing/opentracing-go"

	"github.com/avast/retry-go"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/olivere/elastic"
	"github.com/pkg/errors"
)

const (
	maxRetryAttempts = 4
	maxBulkSize      = 100
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
		logger:        reader.logger,
		sourceFn:      getSourceFn(),
		reader:        reader,
		spanConverter: dbmodel.NewToDomain(objects.TagDotReplacementCharacter),
	}
}

func (finder *TraceFinder) traceIDsMultiSearchRequestBody(traceIDs []model.TraceID, nextTime uint64, endTime time.Time, searchAfterTime map[model.TraceID]uint64) string {
	multiSearchBody := ""
	for _, traceID := range traceIDs {
		finder.logger.Debug(fmt.Sprintf("creating request for trace %s", traceID.String()))
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
		// add search {}\n to prefix and \n to suffix of the search request to match it to multiSearch format
		multiSearchBody = fmt.Sprintf("%s{}\n%s\n", multiSearchBody, requestBody)
	}
	return multiSearchBody
}

func (finder *TraceFinder) getTracesToChannel(traceIDs []model.TraceID, tracesChan chan *model.Trace, startTime time.Time, endTime time.Time) error {
	nextTime := model.TimeAsEpochMicroseconds(startTime)
	searchAfterTime := make(map[model.TraceID]uint64)
	totalDocumentsFetched := make(map[model.TraceID]int)
	tracesMap := make(map[model.TraceID]*model.Trace)
	for {
		if len(traceIDs) == 0 {
			break
		}
		multiSearchBody := finder.traceIDsMultiSearchRequestBody(traceIDs, nextTime, endTime, searchAfterTime)
		// set traceIDs to empty
		traceIDs = nil

		results, err := finder.reader.getMultiSearchResult(multiSearchBody)
		if err != nil || results.Responses == nil || len(results.Responses) == 0 {
			if err != nil {
				return err
			}
			break
		}

		for _, result := range results.Responses {
			if result.Hits == nil || len(result.Hits.Hits) == 0 {
				tracesChan <- nil
				continue
			}
			spans, err := finder.collectSpans(result.Hits.Hits)
			if err != nil {
				finder.logger.Warn(fmt.Sprintf("can't collect spans form result: %s", err.Error()))
				tracesChan <- nil
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
	finder.logger.Debug(fmt.Sprintf("%d traces return", len(tracesMap)))
	if len(tracesMap) == 0 {
		return errors.New(fmt.Sprintf("No search results for traces: %v", traceIDs))
	}
	for traceID := range tracesMap {
		tracesChan <- tracesMap[traceID]
	}
	return nil
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

func (finder *TraceFinder) bulkSearchWithRetry(traceIDs []model.TraceID, tracesChan chan *model.Trace, startTime, endTime time.Time, bulkIndex int) {
	err := retry.Do( // retry in case one of the bulk requests failed
		func() error {
			finder.logger.Debug(fmt.Sprintf("processing bulk %v", bulkIndex))
			return finder.getTracesToChannel(traceIDs, tracesChan, startTime, endTime)
		},
		retry.Attempts(maxRetryAttempts),
		retry.Delay(time.Millisecond*500),
		retry.OnRetry(
			func(n uint, err error) {
				finder.logger.Debug(fmt.Sprintf("retrying bulk %d retry %d/%d", bulkIndex, n+1, maxRetryAttempts))
			}),
	)
	if err != nil {
		finder.logger.Error(fmt.Sprintf("failed to fetch bulk %d with %d traces: %s", bulkIndex, len(traceIDs), err.Error()))
	}
}

func (finder *TraceFinder) multiRead(traceIDs []model.TraceID, startTime, endTime time.Time) ([]*model.Trace, error) {
	if len(traceIDs) == 0 {
		return []*model.Trace{}, nil
	}
	tracesChan := make(chan *model.Trace)
	requestBulksCount := int(math.Ceil(float64(len(traceIDs)) / float64(maxBulkSize)))
	finder.logger.Debug(fmt.Sprintf("performing %v bulk searches for %v traceIDs", requestBulksCount, len(traceIDs)))
	expectedTraceCount := len(traceIDs)
	for i := 0; i < requestBulksCount; i++ {
		bulkStartOffset := i * maxBulkSize
		bulkEnd := int(math.Min(float64(bulkStartOffset+maxBulkSize), float64(len(traceIDs))))
		go finder.bulkSearchWithRetry(traceIDs[bulkStartOffset:bulkEnd], tracesChan, startTime, endTime, i+1)
		time.Sleep(time.Millisecond * 300)
	}

	var traces []*model.Trace
	timeout := false
	for i := 0; i < expectedTraceCount && !timeout; i++ {
		select {
		case trace := <-tracesChan:
			if trace != nil {
				traces = append(traces, trace)
			} else {
				finder.logger.Warn("missing a trace...")
			}
		case <-time.After(15 * time.Second): // continue if there are no traces in the channel for 15 seconds
			{
				finder.logger.Warn("got timeout while waiting for response")
				timeout = true
			}
		}
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
		return nil, errors.Wrap(err, "can't create search request for trace query")
	}
	requestBody = fmt.Sprintf("{}\n%s\n", requestBody)
	searchResult, err := finder.reader.getSearchResult(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "Search service failed")
	}

	bucket, found := searchResult.Aggregations.Terms(traceIDAggregation)
	if !found {
		return nil, ErrUnableToFindTraceIDAggregation
	}

	traceIDBuckets := bucket.Buckets
	return bucketToStringArray(traceIDBuckets)
}

func (finder *TraceFinder) buildTagQuery(k string, v string) elastic.Query {
	objectTagListLen := len(objectTagFieldList)
	queries := make([]elastic.Query, objectTagListLen)
	kd := finder.spanConverter.ReplaceDot(k)
	for i := range objectTagFieldList {
		queries[i] = buildObjectQuery(objectTagFieldList[i], kd, v)
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
