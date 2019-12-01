package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/olivere/elastic"
	"github.com/opentracing/opentracing-go"

	"github.com/pkg/errors"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

const (
	traceIDAggregation = "traceIDs"

	traceIDField           = "traceID"
	durationField          = "duration"
	startTimeField         = "startTime"
	httpPost               = "POST"
	apiTokenHeader         = "X-API-TOKEN"
	serviceNameField       = "process.serviceName"
	operationNameField     = "operationName"
	objectTagsField        = "tag"
	objectProcessTagsField = "process.tag"
	nestedTagsField        = "tags"
	nestedProcessTagsField = "process.tags"
	nestedLogFieldsField   = "logs.fields"
	tagKeyField            = "key"
	tagValueField          = "value"

	defaultDocCount          = 10000 // the default elasticsearch allowed limit
	logzioMaxAggregationSize = 1000
	defaultNumTraces         = 100
)

var (
	// ErrServiceNameNotSet occurs when attempting to query with an empty service name
	ErrServiceNameNotSet = errors.New("Service Name must be set")

	// ErrStartTimeMinGreaterThanMax occurs when start time min is above start time max
	ErrStartTimeMinGreaterThanMax = errors.New("Start Time Minimum is above Maximum")

	// ErrDurationMinGreaterThanMax occurs when duration min is above duration max
	ErrDurationMinGreaterThanMax = errors.New("Duration Minimum is above Maximum")

	// ErrMalformedRequestObject occurs when a request object is nil
	ErrMalformedRequestObject = errors.New("Malformed request object")

	// ErrStartAndEndTimeNotSet occurs when start time and end time are not set
	ErrStartAndEndTimeNotSet = errors.New("Start and End Time must be set")

	// ErrUnableToFindTraceIDAggregation occurs when an aggregation query for TraceIDs fail.
	ErrUnableToFindTraceIDAggregation = errors.New("Could not find aggregation of traceIDs")

	defaultMaxDuration = model.DurationAsMicroseconds(time.Hour * 24)

	objectTagFieldList = []string{objectTagsField, objectProcessTagsField}

	nestedTagFieldList = []string{nestedTagsField, nestedProcessTagsField, nestedLogFieldsField}
)

// LogzioSpanReader is a struct which holds logzio span reader properties
type LogzioSpanReader struct {
	apiToken                string
	apiURL                  string
	logger                  hclog.Logger
	sourceFn                sourceFn
	client                  *http.Client
	traceFinder             TraceFinder
	serviceOperationStorage *ServiceOperationStorage
}

// NewLogzioSpanReader creates a new logzio span reader
func NewLogzioSpanReader(config LogzioConfig, logger hclog.Logger) *LogzioSpanReader {
	reader := &LogzioSpanReader{
		logger:   logger,
		apiToken: config.APIToken,
		apiURL:   config.APIURL(),
		sourceFn: getSourceFn(),
	}
	reader.serviceOperationStorage = NewServiceOperationStorage(reader)
	reader.traceFinder = NewTraceFinder(reader)
	return reader
}

type sourceFn func(query elastic.Query, nextTime uint64) *elastic.SearchSource

func getSourceFn() sourceFn {
	return func(query elastic.Query, nextTime uint64) *elastic.SearchSource {
		searchSource := elastic.NewSearchSource().
			Query(query).
			Size(10000)
		//TerminateAfter(10000)
		searchSource.Sort("startTime", true)
		//SearchAfter(nextTime)
		return searchSource
	}
}

// GetTrace returns a Jaeger trace by traceID
func (reader *LogzioSpanReader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetTrace")
	defer span.Finish()
	currentTime := time.Now()
	traces, err := reader.traceFinder.multiRead(ctx, []model.TraceID{traceID}, currentTime.Add(-time.Hour*48), currentTime)
	if err != nil {
		return nil, err
	}
	if len(traces) == 0 {
		return nil, spanstore.ErrTraceNotFound
	}
	return traces[0], nil
}

// GetServices returns an array of all the service named that are being monitored
func (reader *LogzioSpanReader) GetServices(ctx context.Context) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetServices")
	defer span.Finish()
	return reader.serviceOperationStorage.getServices(ctx)
}

// GetOperations returns an array of all the operation a specific service performed
func (reader *LogzioSpanReader) GetOperations(ctx context.Context, service string) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetOperations")
	defer span.Finish()
	return reader.serviceOperationStorage.getOperations(ctx, service)
}

// FindTraces return an array of Jaeger traces by a search query
func (reader *LogzioSpanReader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FindTraces")
	defer span.Finish()

	uniqueTraceIDs, err := reader.FindTraceIDs(ctx, query)
	if err != nil {
		return nil, err
	}
	return reader.traceFinder.multiRead(ctx, uniqueTraceIDs, query.StartTimeMin, query.StartTimeMax)
}

// FindTraceIDs returns an array of traceIds by a search query
func (reader *LogzioSpanReader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FindTraceIDs")
	defer span.Finish()

	if err := validateQuery(query); err != nil {
		return nil, err
	}
	if query.NumTraces == 0 {
		query.NumTraces = defaultNumTraces
	}
	esTraceIDs, err := reader.traceFinder.findTraceIDsStrings(ctx, query)
	if err != nil {
		return nil, err
	}
	reader.logger.Error(fmt.Sprint(esTraceIDs))
	return convertTraceIDsStringsToModels(esTraceIDs)
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
	return nil
}

func (reader *LogzioSpanReader) getMultiSearchResult(requestBody string) (elastic.MultiSearchResult, error) {
	if reader.apiToken == "" {
		return elastic.MultiSearchResult{}, errors.New("empty API token, can't perform search")
	}
	client := http.Client{}
	req, err := http.NewRequest(httpPost, reader.apiURL, strings.NewReader(requestBody))
	if err != nil {
		reader.logger.Error("failed to create multiSearch request")
		return elastic.MultiSearchResult{}, err
	}
	req.Header.Add(apiTokenHeader, reader.apiToken)
	resp, err := client.Do(req)
	if err != nil {
		reader.logger.Error("failed to execute multiSearch request")
		return elastic.MultiSearchResult{}, err
	}

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		reader.logger.Error("can't read response body")
		return elastic.MultiSearchResult{}, err
	}
	reader.logger.Error(string(responseBytes))
	if err := resp.Body.Close(); err != nil {
		reader.logger.Warn("can't close response body, possible memory leak")
	}
	reader.logger.Debug(fmt.Sprintf("got response from logz.io: %s", string(responseBytes)))

	var multiSearchResult elastic.MultiSearchResult
	if err := json.Unmarshal(responseBytes, &multiSearchResult); err != nil {
		return elastic.MultiSearchResult{}, errors.Wrap(err, "failed to parse http response")
	}
	return multiSearchResult, err
}

// GetDependencies returns an array of all the dependencies in a specific time range
func (*LogzioSpanReader) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}
