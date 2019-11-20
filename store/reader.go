package store

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jaegertracing/jaeger/pkg/es"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/olivere/elastic"
	"github.com/opentracing/opentracing-go"
	ottag "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

// LogzioSpanReader is a struct which holds logzio span reader properties
type LogzioSpanReader struct {
	apiToken		string
	logger			hclog.Logger
	sourceFn		sourceFn
	client 			es.Client
	spanConverter	dbmodel.ToDomain
}

// NewLogzioSpanReader creates a new logzio span reader
func NewLogzioSpanReader(config LogzioConfig, logger hclog.Logger) *LogzioSpanReader {
	return &LogzioSpanReader{
		logger:   logger,
		apiToken: config.APIToken,
	}
}

type sourceFn func(query elastic.Query, nextTime uint64) *elastic.SearchSource

// GetTrace returns a Jaeger trace by traceID
func (reader *LogzioSpanReader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetTrace")
	defer span.Finish()
	currentTime := time.Now()
	traces, err := reader.multiRead(ctx, []model.TraceID{traceID}, currentTime.Add(-time.Hour*24), currentTime)
	if err != nil {
		return nil, err
	}
	if len(traces) == 0 {
		return nil, spanstore.ErrTraceNotFound
	}
	return traces[0], nil
}

func (reader *LogzioSpanReader) collectSpans(esSpansRaw []*elastic.SearchHit) ([]*model.Span, error) {
	spans := make([]*model.Span, len(esSpansRaw))

	for i, esSpanRaw := range esSpansRaw {
		jsonSpan, err := reader.unmarshalJSONSpan(esSpanRaw)
		if err != nil {
			return nil, errors.Wrap(err, "Marshalling JSON to span object failed")
		}
		dbSpan := jsonSpan.transformToDbModelSpan()
		span, err := reader.spanConverter.SpanToDomain(dbSpan)
		if err != nil {
			return nil, errors.Wrap(err, "Converting JSONSpan to domain Span failed")
		}
		spans[i] = span
	}
	return spans, nil
}

func (reader *LogzioSpanReader) unmarshalJSONSpan(esSpanRaw *elastic.SearchHit) (*logzioSpan, error) {
	esSpanInByteArray := esSpanRaw.Source

	var jsonSpan logzioSpan

	decoder := json.NewDecoder(bytes.NewReader(*esSpanInByteArray))
	decoder.UseNumber()
	if err := decoder.Decode(&jsonSpan); err != nil {
		return nil, err
	}
	return &jsonSpan, nil
}

// GetServices returns an array of all the service named that are being monitored
func (*LogzioSpanReader) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}

// GetOperations returns an array of all the operation a specific service performed
func (*LogzioSpanReader) GetOperations(ctx context.Context, service string) ([]string, error) {
	return nil, nil
}

// FindTraces return an array of Jaeger traces by a search query
func (*LogzioSpanReader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	return nil, nil
}

// FindTraceIDs returns an array of traceIds by a search query
func (*LogzioSpanReader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	return nil, nil
}

func (reader *LogzioSpanReader) multiRead(ctx context.Context, traceIDs []model.TraceID, startTime, endTime time.Time) ([]*model.Trace, error) {

	childSpan, _ := opentracing.StartSpanFromContext(ctx, "multiRead")
	childSpan.LogFields(otlog.Object("trace_ids", traceIDs))
	defer childSpan.Finish()

	if len(traceIDs) == 0 {
		return []*model.Trace{}, nil
	}

	nextTime := model.TimeAsEpochMicroseconds(startTime)

	searchAfterTime := make(map[model.TraceID]uint64)
	totalDocumentsFetched := make(map[model.TraceID]int)
	tracesMap := make(map[model.TraceID]*model.Trace)
	for {
		if len(traceIDs) == 0 {
			break
		}
		searchRequests := make([]*elastic.SearchRequest, len(traceIDs))
		for i, traceID := range traceIDs {
			query := elastic.NewTermQuery("traceID", traceID.String())
			if val, ok := searchAfterTime[traceID]; ok {
				nextTime = val
			}

			reader := reader.sourceFn(query, nextTime)

			searchRequests[i] = elastic.NewSearchRequest().
				IgnoreUnavailable(true).
				Source(reader)
		}
		// set traceIDs to empty
		traceIDs = nil
		results, err := reader.client.MultiSearch().Add(searchRequests...).Do(ctx)
		if err != nil {
			logErrorToSpan(childSpan, err)
			return nil, err
		}

		if results.Responses == nil || len(results.Responses) == 0 {
			break
		}

		for _, result := range results.Responses {
			if result.Hits == nil || len(result.Hits.Hits) == 0 {
				continue
			}
			spans, err := reader.collectSpans(result.Hits.Hits)
			if err != nil {
				logErrorToSpan(childSpan, err)
				return nil, err
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

	var traces []*model.Trace
	for _, trace := range tracesMap {
		traces = append(traces, trace)
	}
	return traces, nil
}

// GetDependencies returns an array of all the dependencies in a specific time range
func (*LogzioSpanReader) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}

//func (sr *LogzioSpanReader) multiRead(ctx context.Context, traceIDs []model.TraceID, startTime, endTime time.Time) ([]*model.Trace, error) {
//
//	childSpan, _ := opentracing.StartSpanFromContext(ctx, "multiRead")
//	childSpan.LogFields(otlog.Object("trace_ids", traceIDs))
//	defer childSpan.Finish()
//
//	if len(traceIDs) == 0 {
//		return []*model.Trace{}, nil
//	}
//
//	// Add an hour in both directions so that traces that straddle two indexes are retrieved.
//	// i.e starts in one and ends in another.
//	nextTime := model.TimeAsEpochMicroseconds(startTime.Add(-time.Hour))
//
//	searchAfterTime := make(map[model.TraceID]uint64)
//	totalDocumentsFetched := make(map[model.TraceID]int)
//	tracesMap := make(map[model.TraceID]*model.Trace)
//	for {
//		if len(traceIDs) == 0 {
//			break
//		}
//		searchRequests := make([]*elastic.SearchRequest, len(traceIDs))
//		for i, traceID := range traceIDs {
//			query := elastic.NewTermQuery("traceID", traceID.String())
//			if val, ok := searchAfterTime[traceID]; ok {
//				nextTime = val
//			}
//
//			s := sr.sourceFn(query, nextTime)
//
//			searchRequests[i] = elastic.NewSearchRequest().
//				IgnoreUnavailable(true).
//				Type(spanType).
//				Source(s)
//		}
//		// set traceIDs to empty
//		traceIDs = nil
//		results, err := sr.client.MultiSearch().Add(searchRequestreader...).Index(indicereader...).Do(sr.ctx)
//
//		if err != nil {
//			logErrorToSpan(childSpan, err)
//			return nil, err
//		}
//
//		if results.Responses == nil || len(results.Responses) == 0 {
//			break
//		}
//
//		for _, result := range results.Responses {
//			if result.Hits == nil || len(result.Hits.Hits) == 0 {
//				continue
//			}
//			spans, err := sr.collectSpans(result.Hits.Hits)
//			if err != nil {
//				logErrorToSpan(childSpan, err)
//				return nil, err
//			}
//			lastSpan := spans[len(spans)-1]
//
//			if traceSpan, ok := tracesMap[lastSpan.TraceID]; ok {
//				traceSpan.Spans = append(traceSpan.Spans, spanreader...)
//			} else {
//				tracesMap[lastSpan.TraceID] = &model.Trace{Spans: spans}
//			}
//
//			totalDocumentsFetched[lastSpan.TraceID] = totalDocumentsFetched[lastSpan.TraceID] + len(result.Hits.Hits)
//			if totalDocumentsFetched[lastSpan.TraceID] < int(result.TotalHits()) {
//				traceIDs = append(traceIDs, lastSpan.TraceID)
//				searchAfterTime[lastSpan.TraceID] = model.TimeAsEpochMicroseconds(lastSpan.StartTime)
//			}
//		}
//	}
//
//	var traces []*model.Trace
//	for _, trace := range tracesMap {
//		traces = append(traces, trace)
//	}
//	return traces, nil
//}

func logErrorToSpan(span opentracing.Span, err error) {
	ottag.Error.Set(span, true)
	span.LogFields(otlog.Error(err))
}