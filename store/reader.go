package store

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/olivere/elastic"
	"github.com/opentracing/opentracing-go"

	//ottag "github.com/opentracing/opentracing-go/ext"
	//otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)
const (
	spanIndex               = "jaeger-span-"
	serviceIndex            = "jaeger-service-"
	archiveIndexSuffix      = "archive"
	archiveReadIndexSuffix  = archiveIndexSuffix + "-read"
	archiveWriteIndexSuffix = archiveIndexSuffix + "-write"
	traceIDAggregation      = "traceIDs"
	indexPrefixSeparator    = "-"

	traceIDField           = "traceID"
	durationField          = "duration"
	startTimeField         = "startTime"
	httpPost       = "POST"
	apiTokenHeader = "X-API-TOKEN"
	serviceNameField       = "process.serviceName"
	operationNameField     = "operationName"
	objectTagsField        = "tag"
	objectProcessTagsField = "process.tag"
	nestedTagsField        = "tags"
	nestedProcessTagsField = "process.tags"
	nestedLogFieldsField   = "logs.fields"
	tagKeyField            = "key"
	tagValueField          = "value"

	defaultDocCount  = 10000 // the default elasticsearch allowed limit
	defaultNumTraces = 100
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
	apiToken    string
	logger      hclog.Logger
	sourceFn    sourceFn
	client      *http.Client
	traceFinder TraceFinder
}

// NewLogzioSpanReader creates a new logzio span reader
func NewLogzioSpanReader(config LogzioConfig, logger hclog.Logger) *LogzioSpanReader {
	return &LogzioSpanReader{
		logger:      logger,
		apiToken:    config.APIToken,
		sourceFn:    getSourceFn(),
		traceFinder: NewTraceFinder(config.APIToken, logger),
	}
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
	traces, err := reader.traceFinder.multiRead(ctx, []model.TraceID{traceID}, currentTime.Add(-time.Hour*240), currentTime)
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
	return []string{"frontend"}, nil
}

// GetOperations returns an array of all the operation a specific service performed
func (reader *LogzioSpanReader) GetOperations(ctx context.Context, service string) ([]string, error) {
	reader.logger.Error("GGGGGGGGet operations called")

	return nil, nil
}

// FindTraces return an array of Jaeger traces by a search query
func (reader *LogzioSpanReader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	reader.logger.Error("FFFFFFFFFFFFFFFFFFFindTraces called")
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
	reader.logger.Error("FFFFFFFFFFFFFFFFFFFindTraceIds called")
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

func parseHTTPResponse(resp *http.Response, logger hclog.Logger) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("can't read response body")
		return nil, err
	}
	logger.Error(string(body))
	if err := resp.Body.Close(); err != nil {
		logger.Warn("can't close response body, possible memory leak")
	}
	logger.Debug(fmt.Sprintf("got response from logz.io: %s", string(body)))
	return body, err
}

func getHTTPResponseBytes(requestBody string, apiToken string, logger hclog.Logger) (*http.Response, error) {
	client := http.Client{}
	req, err := http.NewRequest(httpPost, "https://api-eu.logz.io/v1/elasticsearch/_msearch", strings.NewReader(requestBody))
	if err != nil {
		logger.Error("failed to create multiSearch request")
		return nil, err
	}
	req.Header.Add(apiTokenHeader, apiToken)
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to execute multiSearch request" + err.Error())
		return nil, err
	}
	return resp, err
}

// GetDependencies returns an array of all the dependencies in a specific time range
func (*LogzioSpanReader) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}
