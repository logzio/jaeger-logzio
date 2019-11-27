package store

import (
	"fmt"
	"time"

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
}


// NewTraceFinder creates trace finder object
func NewTraceFinder(logger hclog.Logger) TraceFinder {
	return TraceFinder{
		logger:   logger,
		sourceFn: getSourceFn(),
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

func (finder *TraceFinder) getTracesMap(apiToken string, traceIDs []model.TraceID, startTime time.Time, endTime time.Time) map[model.TraceID]*model.Trace {
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
		resp, err := getHTTPResponse(multiSearchBody, apiToken, finder.logger)
		if err != nil {
			break
		}
		results, err := parseMultiSearchResponse(resp, finder.logger)
		if err != nil {
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
