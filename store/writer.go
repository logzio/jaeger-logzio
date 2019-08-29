package store

import (
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
)

type logzioSpanWriter struct {
	accountToken string
	logger   hclog.Logger
}

func (*logzioSpanWriter) WriteSpan(span *model.Span) error {
	panic("implement me")
}

func NewLogzioSpanWriter(accountToken string, logger hclog.Logger) *logzioSpanWriter {
	w := &logzioSpanWriter{
		accountToken:	accountToken,
		logger: logger,
	}
	return w
}