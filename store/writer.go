package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/logzio/jaeger-logzio/store/objects"

	"github.com/logzio/logzio-go"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
)

const (
	dropLogsDiskThreshold = 98
)

type loggerWriter struct {
	logger hclog.Logger
}

//this is to convert between jaeger log messages and logzioSender log messages
func (writer *loggerWriter) Write(msgBytes []byte) (n int, err error) {
	msgString := string(msgBytes)
	if strings.Contains(strings.ToLower(msgString), "error") {
		writer.logger.Error(msgString)
	} else {
		writer.logger.Debug(msgString)
	}
	return len(msgBytes), nil
}

// LogzioSpanWriter is a struct which holds logzio span writer properties
type LogzioSpanWriter struct {
	accountToken string
	logger       hclog.Logger
	sender       *logzio.LogzioSender
	serviceCache cache.Cache
}

// NewLogzioSpanWriter creates a new logzio span writer for jaeger
func NewLogzioSpanWriter(config LogzioConfig, logger hclog.Logger) (*LogzioSpanWriter, error) {
	sender, err := logzio.New(
		config.AccountToken,
		logzio.SetUrl(config.ListenerURL()),
		logzio.SetDebug(&loggerWriter{logger: logger}),
		logzio.SetDrainDiskThreshold(dropLogsDiskThreshold))

	if err != nil {
		return nil, err
	}
	spanWriter := &LogzioSpanWriter{
		accountToken: config.AccountToken,
		logger:       logger,
		sender:       sender,
		serviceCache: cache.NewLRUWithOptions(
			100000,
			&cache.Options{
				TTL: 24 * time.Hour,
			},
		),
	}
	return spanWriter, nil
}

// WriteSpan receives a Jaeger span, converts it to logzio span and sends it to logzio
func (spanWriter *LogzioSpanWriter) WriteSpan(ctx context.Context, span *model.Span) error{
	span.Tags = spanWriter.dropEmptyTags(span.Tags)
	span.Process.Tags = spanWriter.dropEmptyTags(span.Process.Tags)
	spanBytes, err := objects.TransformToLogzioSpanBytes(span)
	if err != nil {
		return err
	}
	err = spanWriter.sender.Send(spanBytes)
	if err != nil {
		return err
	}
	service := objects.NewLogzioService(span)
	serviceHash, err := service.HashCode()

	if spanWriter.serviceCache.Get(serviceHash) == nil || err != nil {
		if err == nil {
			spanWriter.serviceCache.Put(serviceHash, serviceHash)
		}
		serviceBytes, err := json.Marshal(service)
		if err != nil {
			return err
		}
		err = spanWriter.sender.Send(serviceBytes)
	}
	return err
}

// Close stops and drains logzio sender
func (spanWriter *LogzioSpanWriter) Close() {
	spanWriter.sender.Stop()
}

func (spanWriter *LogzioSpanWriter) dropEmptyTags(tags []model.KeyValue) []model.KeyValue {
	for i, tag := range tags {
		if tag.Key == "" {
			tags[i] = tags[len(tags)-1] // Copy last element to index i.
			tags = tags[:len(tags)-1]   // Truncate slice.
			spanWriter.logger.Warn(fmt.Sprintf("Found tag empty key: %s, dropping tag..", tag.String()))
		}
	}
	return tags
}
