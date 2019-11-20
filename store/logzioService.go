package store

import (
	"fmt"
	"hash/fnv"

	"github.com/jaegertracing/jaeger/model"
)

const serviceLogType = "jaegerService"

//LogzioService type, for query purposes
type LogzioService struct {
	OperationName string `json:"operationName"`
	ServiceName   string `json:"serviceName"`
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

func (service *LogzioService) hashCode() (string, error) {
	h := fnv.New64a()
	_, err := h.Write(append([]byte(service.ServiceName), []byte(service.OperationName)...))
	return fmt.Sprintf("%x", h.Sum64()), err
}
