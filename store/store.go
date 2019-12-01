package store

import (
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

// Store is span store struct for logzio jaeger span storage
type Store struct {
	reader *LogzioSpanReader
	writer *LogzioSpanWriter
}

// NewLogzioStore creates a new logzio span store for jaeger
func NewLogzioStore(config LogzioConfig, logger hclog.Logger) *Store {
	reader := NewLogzioSpanReader(config, logger)
	if config.APIToken == "" {
		logger.Warn( "No api token found, can't create span reader")
	}
	if config.AccountToken == "" {
		logger.Warn( "No account token found, spans will not be saved")
	}
	writer, err := NewLogzioSpanWriter(config, logger)
	if err != nil {
		logger.Error("Failed to create logzio span writer: " + err.Error())
	}
	store := &Store{
		reader: reader,
		writer: writer,
	}
	return store
}

// Close the span store
func (store *Store) Close() {
	store.writer.Close()
}

// SpanReader returns the created logzio span reader
func (store *Store) SpanReader() spanstore.Reader {
	return store.reader
}

// SpanWriter returns the created logzio span writer
func (store *Store) SpanWriter() spanstore.Writer {
	return store.writer
}

// DependencyReader return the created logzio dependency store
func (store *Store) DependencyReader() dependencystore.Reader {
	return store.reader
}
