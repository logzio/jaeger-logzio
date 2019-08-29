package store

import (
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var (
	_ shared.StoragePlugin = (*Store)(nil)
)

type Store struct {
	reader *logzioSpanReader
	writer *logzioSpanWriter
}

func NewLogzioStore(accountToken string, apiToken string , logger hclog.Logger) *Store {

	reader := NewLogzioSpanReader(apiToken, logger)


	writer := NewLogzioSpanWriter(accountToken, logger)

	store := &Store{
		reader: reader,
		writer: writer,
	}

	return store
}

//func (s *Store) Close() error {
//	return s.writer.Close()
//}

func (store *Store) SpanReader() spanstore.Reader {
	return store.reader
}

func (store *Store) SpanWriter() spanstore.Writer {
	return store.writer
}

func (store *Store) DependencyReader() dependencystore.Reader {
	return store.reader
}


