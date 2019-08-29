package store

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"time"
)

type logzioSpanReader struct {
	apiToken string
	logger   hclog.Logger
}

func NewLogzioSpanReader(accountToken string, logger hclog.Logger) *logzioSpanReader {
	return &logzioSpanReader{
		logger:   logger,
		apiToken: accountToken,
	}
}

func (*logzioSpanReader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	panic("implement me")
}

func (*logzioSpanReader) GetServices(ctx context.Context) ([]string, error) {
	panic("implement me")
}

func (*logzioSpanReader) GetOperations(ctx context.Context, service string) ([]string, error) {
	panic("implement me")
}

func (*logzioSpanReader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	panic("implement me")
}

func (*logzioSpanReader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	panic("implement me")
}

func (*logzioSpanReader) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	panic("implement me")
}