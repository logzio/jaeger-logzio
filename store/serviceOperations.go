package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/olivere/elastic"
	"github.com/pkg/errors"
)

const (
	serviceName = "serviceName"
	operationsAggregation = "distinct_operations"
	servicesAggregation   = "distinct_services"
	logzioMaxAggregationSize = 1000

)

// ServiceOperationStorage stores service to operation pairs.
type ServiceOperationStorage struct {
	client       elastic.Client
	logger       hclog.Logger
	serviceCache cache.Cache
	apiToken	 string
}

// NewServiceOperationStorage returns a new ServiceOperationStorage.
func NewServiceOperationStorage(logger hclog.Logger,	apiToken string,) *ServiceOperationStorage {
	return &ServiceOperationStorage{
		client: elastic.Client{},
		logger: logger,
		apiToken: apiToken,
	}
}


func (soStorage *ServiceOperationStorage) getServices(context context.Context) ([]string, error) {
	serviceAggregation := getServicesAggregation()

	//searchService := soStorage.client.Search().
	//	Size(0). // set to 0 because we don't want actual documents.
	//	IgnoreUnavailable(true).
	//	Aggregation(servicesAggregation, serviceAggregation)

	searchBody, err := elastic.NewSearchRequest().
		Size(0).
		IgnoreUnavailable(true).
		Aggregation(servicesAggregation, serviceAggregation).Body()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create search service request")
	}
	searchBody = fmt.Sprintf("{}\n%s\n", searchBody)
	soStorage.logger.Error(searchBody)
	httpResponse, err := getHTTPResponseBytes(searchBody, soStorage.apiToken, soStorage.logger)
	if err != nil {
		return nil, err
	}

    responseBytes, err := parseHTTPResponse(httpResponse, soStorage.logger)
	if err != nil {
		return nil, err
	}

	soStorage.logger.Error(string(responseBytes))

    var multiSearchResult elastic.MultiSearchResult
	if err := json.Unmarshal(responseBytes, &multiSearchResult); err != nil {
		return nil, errors.Wrap(err, "failed to parse http response")
	}

	searchResult := multiSearchResult.Responses[0]
	if searchResult.Aggregations == nil {
		return []string{}, nil
	}
	bucket, found := searchResult.Aggregations.Terms(servicesAggregation)
	if !found {
		return nil, errors.New("Could not find aggregation of " + servicesAggregation)
	}
	serviceNamesBucket := bucket.Buckets
	return bucketToStringArray(serviceNamesBucket)
}

func getServicesAggregation() elastic.Query {
	return elastic.NewTermsAggregation().
		Field(serviceName).
		Size(logzioMaxAggregationSize) // Must set to some large number. ES deprecated size omission for aggregating all. https://github.com/elastic/elasticsearch/issues/18838
}