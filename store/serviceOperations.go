package store

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/olivere/elastic"
	"github.com/pkg/errors"
)

const (
	serviceName = "serviceName"
)

// ServiceOperationStorage stores service to operation pairs.
type ServiceOperationStorage struct {
	logger       hclog.Logger
	serviceCache cache.Cache
	reader       *LogzioSpanReader
}

// NewServiceOperationStorage returns a new ServiceOperationStorage.
func NewServiceOperationStorage(reader *LogzioSpanReader) *ServiceOperationStorage {
	return &ServiceOperationStorage{
		reader: reader,
		logger: reader.logger,
	}
}

func (soStorage *ServiceOperationStorage) getServices(context context.Context) ([]string, error) {
	return soStorage.getUniqueValues(context, serviceName, nil)
}

func (soStorage *ServiceOperationStorage) getOperations(context context.Context, service string) ([]string, error) {
	serviceQuery := elastic.NewTermQuery(serviceName, service)
	return soStorage.getUniqueValues(context, operationNameField, serviceQuery)
}

func getAggregation(field string) elastic.Query {
	return elastic.NewTermsAggregation().
		Field(field).
		Size(logzioMaxAggregationSize)
}

func (soStorage *ServiceOperationStorage) getUniqueValues(context context.Context, field string, termsQuery elastic.Query) ([]string, error) {
	serviceFilter := getAggregation(field)
	aggregationString := "distinct_" + field

	searchRequest := elastic.NewSearchRequest().
		Size(0).
		IgnoreUnavailable(true).
		Aggregation(aggregationString, serviceFilter)

	if termsQuery != nil {
		searchRequest = searchRequest.Query(termsQuery)
	}
	searchBody, err := searchRequest.Body()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create search service request")
	}
	searchBody = fmt.Sprintf("{}\n%s\n", searchBody)

	searchResult, err := soStorage.reader.getSearchResult(searchBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute search service request")
	}
	if searchResult.Aggregations == nil {
		return []string{}, nil
	}
	bucket, found := searchResult.Aggregations.Terms(aggregationString)
	if !found {
		return nil, errors.New("Could not find aggregation of " + aggregationString)
	}
	operationNamesBucket := bucket.Buckets
	return bucketToStringArray(operationNamesBucket)
}
