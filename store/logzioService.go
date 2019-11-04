package store

import (
	"fmt"
	"hash/fnv"

	"github.com/jaegertracing/jaeger/model"
)

const serviceLogType = "jaegerService"

//LogzioService type, for query purposes
type LogzioService struct {
	OperationName string `json:"serviceName"`
	ServiceName   string `json:"operationName"`
	Type          string `json:"type"`
}

//NewLogzioService creates a new logzio service from a span
func NewLogzioService(span *model.Span) LogzioService {
	service := LogzioService{
		ServiceName:   span.Process.ServiceName,
		OperationName: span.OperationName,
		Type:          serviceLogType,
	}
	return service
}

func (service *LogzioService) hashCode() string {
	h := fnv.New64a()
	h.Write([]byte(service.ServiceName))
	h.Write([]byte(service.OperationName))
	return fmt.Sprintf("%x", h.Sum64())
}
