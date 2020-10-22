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
	objectTagsField        = "JaegerTag"
	objectProcessTagsField = "process.tag"
	tagKeyField            = "key"
	tagValueField          = "value"

	defaultDocCount          = 10000 // the default elasticsearch allowed limit
	logzioMaxAggregationSize = 1000
	defaultNumTraces         = 100
	maxSearchWindowHours     = 48
	singleValueIndex         = 0
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
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		},
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
			Size(defaultDocCount)
		//TerminateAfter(10000)
		searchSource.Sort(startTimeField, true)
		//SearchAfter(nextTime)
		return searchSource
	}
}

// GetTrace returns a Jaeger trace by traceID
func (reader *LogzioSpanReader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetTrace")
	defer span.Finish()
	currentTime := time.Now()
	traces, err := reader.traceFinder.multiRead([]model.TraceID{traceID}, currentTime.Add(-time.Hour*maxSearchWindowHours), currentTime)
	if err != nil {
		return nil, err
	}
	if len(traces) == 0 {
		return nil, spanstore.ErrTraceNotFound
	}
	//here we are using multiread to get a single trace. since multiread returns an array of result, we only want the first (and only) result
	return traces[singleValueIndex], nil
}

// GetServices returns an array of all the service names that are being monitored
func (reader *LogzioSpanReader) GetServices(ctx context.Context) ([]string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetServices")
	defer span.Finish()
	return reader.serviceOperationStorage.getServices(ctx)
}

// GetOperations returns an array of all the operations a specific service performed
func  (reader *LogzioSpanReader) GetOperations(ctx context.Context, query spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetOperations")
	defer span.Finish()
	operations, err := reader.serviceOperationStorage.getOperations(ctx, query.ServiceName)
	if err != nil {
		return nil, err
	}

	var result []spanstore.Operation
	for _, operation := range operations {
		result = append(result, spanstore.Operation{
			Name: operation,
		})
	}
	return result, err


}

// FindTraces return an array of Jaeger traces by a search query
func (reader *LogzioSpanReader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FindTraces")
	defer span.Finish()

	spansMaxMinTimeDifferenceHours := query.StartTimeMax.Sub(query.StartTimeMin).Hours()
	if spansMaxMinTimeDifferenceHours > maxSearchWindowHours {
		query.StartTimeMin = query.StartTimeMax.Add(-time.Hour * maxSearchWindowHours)
	}
	uniqueTraceIDs, err := reader.FindTraceIDs(ctx, query)
	if err != nil {
		return nil, err
	}
	return reader.traceFinder.multiRead(uniqueTraceIDs, query.StartTimeMin, query.StartTimeMax)
}

// FindTraceIDs retrieve traceIDs that match the traceQuery
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
	reader.logger.Debug(fmt.Sprintf("found traceIDs: %v", esTraceIDs))
	return convertTraceIDsStringsToModels(esTraceIDs)
}

func (reader *LogzioSpanReader) getHTTPRequest(requestBody string) (*http.Request, error) {
	reader.logger.Debug("creating multisearch request: %s", requestBody)
	req, err := http.NewRequest(httpPost, reader.apiURL, strings.NewReader(requestBody))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create multiSearch request")
	}
	req.Header.Add(apiTokenHeader, reader.apiToken)
	return req, nil
}

func (reader *LogzioSpanReader) getHTTPResponseBytes(request *http.Request) ([]byte, error) {
	resp, err := reader.client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed perform multiSearch request")
	}

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "can't read response body")
	}
	if err = resp.Body.Close(); err != nil {
		reader.logger.Warn("can't close response body, possible memory leak")
	}
	reader.logger.Trace(fmt.Sprintf("got response from logz.io: %s", string(responseBytes)))

	if err = checkErrorResponse(responseBytes); err != nil {
		return nil, err
	}
	return responseBytes, nil
}

//this is kink of a hack function, we use multisearch to perform a single search
func (reader *LogzioSpanReader) getSearchResult(requestBody string) (*elastic.SearchResult, error) {
	multiSearchResult, err := reader.getMultiSearchResult(requestBody)
	if err != nil {
		return nil, err
	}

	if len(multiSearchResult.Responses) > singleValueIndex {
		return multiSearchResult.Responses[singleValueIndex], nil
	}
	return nil, nil
}

func (reader *LogzioSpanReader) getMultiSearchResult(requestBody string) (elastic.MultiSearchResult, error) {
	if reader.apiToken == "" {
		return elastic.MultiSearchResult{}, errors.New("empty API token, can't perform search")
	}
	req, err := reader.getHTTPRequest(requestBody)
	if err != nil {
		return elastic.MultiSearchResult{}, err
	}
	responseBytes, err := reader.getHTTPResponseBytes(req)
	if err != nil {
		return elastic.MultiSearchResult{}, err
	}

	var multiSearchResult elastic.MultiSearchResult
	if err = json.Unmarshal(responseBytes, &multiSearchResult); err != nil {
		return elastic.MultiSearchResult{}, errors.Wrap(err, "failed to parse http response")
	}
	return multiSearchResult, err
}

// GetDependencies returns an array of all the dependencies in a specific time range
func (*LogzioSpanReader) GetDependencies(ctx context.Context, endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}

func checkErrorResponse(response []byte) error {
	var respMap map[string]interface{}
	_ = json.Unmarshal(response, &respMap)
	_, exist := respMap["errorCode"]
	if exist {
		return errors.New(fmt.Sprintf("got error response: %s", string(response)))
	}
	return nil
}
