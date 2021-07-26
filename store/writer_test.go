package store

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/logzio/jaeger-logzio/store/objects"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
)

const (
	testOperation = "testOperation"
	testService   = "testService"
	testValue     = "testValue"
	testName      = "John Smith"
)

func TestWriteSpan(tester *testing.T) {
	var recordedRequests []byte
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recordedRequests, _ = ioutil.ReadAll(req.Body)
		rw.WriteHeader(http.StatusOK)
	}))

	defer server.Close()
	date, _ := time.Parse(time.RFC3339, "1995-04-21T22:08:41+00:00")
	span := &model.Span{
		TraceID:       model.NewTraceID(0, 1),
		SpanID:        model.NewSpanID(0),
		OperationName: testOperation,
		Process: &model.Process{
			ServiceName: testService,
		},
		StartTime: date,
	}

	writer, _ := NewLogzioSpanWriter(LogzioConfig{AccountToken: testAccountToken, CustomListenerURL: server.URL, Compress: false}, logger)
	assert.NoError(tester, writer.WriteSpan(context.Background(), span))

	time.Sleep(time.Second * 6)
	requests := strings.Split(string(recordedRequests), "\n")
	var logzioSpan objects.LogzioSpan
	assert.NoError(tester, json.Unmarshal([]byte(requests[0]), &logzioSpan))

	assert.Equal(tester, logzioSpan.OperationName, testOperation)
	assert.Equal(tester, logzioSpan.Process.ServiceName, testService)

	var logzioService objects.LogzioService
	assert.NoError(tester, json.Unmarshal([]byte(requests[1]), &logzioService))

	assert.Equal(tester, logzioService.OperationName, testOperation)
	assert.Equal(tester, logzioService.ServiceName, testService)
}
func TestWriteSpanInMemory(tester *testing.T) {
	var recordedRequests []byte
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recordedRequests, _ = ioutil.ReadAll(req.Body)
		rw.WriteHeader(http.StatusOK)
	}))

	defer server.Close()
	date, _ := time.Parse(time.RFC3339, "1995-04-21T22:08:41+00:00")
	span := &model.Span{
		TraceID:       model.NewTraceID(0, 1),
		SpanID:        model.NewSpanID(0),
		OperationName: testOperation,
		Process: &model.Process{
			ServiceName: testService,
		},
		StartTime: date,
	}

	writer, _ := NewLogzioSpanWriter(LogzioConfig{AccountToken: testAccountToken,
		CustomListenerURL: server.URL,
		Compress:          false,
		InMemoryQueue:     true}, logger)
	assert.NoError(tester, writer.WriteSpan(context.Background(), span))

	time.Sleep(time.Second * 6)
	requests := strings.Split(string(recordedRequests), "\n")
	var logzioSpan objects.LogzioSpan
	assert.NoError(tester, json.Unmarshal([]byte(requests[0]), &logzioSpan))

	assert.Equal(tester, logzioSpan.OperationName, testOperation)
	assert.Equal(tester, logzioSpan.Process.ServiceName, testService)

	var logzioService objects.LogzioService
	assert.NoError(tester, json.Unmarshal([]byte(requests[1]), &logzioService))

	assert.Equal(tester, logzioService.OperationName, testOperation)
	assert.Equal(tester, logzioService.ServiceName, testService)
}

func TestDropEmptyTags(tester *testing.T) {
	var recordedRequests []byte
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recordedRequests, _ = ioutil.ReadAll(req.Body)
		rw.WriteHeader(http.StatusOK)
	}))

	defer server.Close()
	tags := []model.KeyValue{
		{
			Key:  "testTag",
			VStr: testValue,
		},
		{
			Key:  "",
			VStr: testValue,
		},
	}
	span := &model.Span{
		TraceID: model.NewTraceID(0, 1),
		SpanID:  model.NewSpanID(0),
		Tags:    tags,
		Process: model.NewProcess(testService, tags),
	}
	writer, _ := NewLogzioSpanWriter(LogzioConfig{AccountToken: testAccountToken,
		CustomListenerURL: server.URL,
		Compress:          false}, logger)
	assert.NoError(tester, writer.WriteSpan(context.Background(), span))

	time.Sleep(time.Second * 6)
	requests := strings.Split(string(recordedRequests), "\n")
	var logzioSpan objects.LogzioSpan
	assert.NoError(tester, json.Unmarshal([]byte(requests[0]), &logzioSpan))
	assert.Equal(tester, 1, len(logzioSpan.Tag))
	assert.Equal(tester, 1, len(logzioSpan.Process.Tag))

}
func TestDropEmptyTagsInMemory(tester *testing.T) {
	var recordedRequests []byte
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recordedRequests, _ = ioutil.ReadAll(req.Body)
		rw.WriteHeader(http.StatusOK)
	}))

	defer server.Close()
	tags := []model.KeyValue{
		{
			Key:  "testTag",
			VStr: testValue,
		},
		{
			Key:  "",
			VStr: testValue,
		},
	}
	span := &model.Span{
		TraceID: model.NewTraceID(0, 1),
		SpanID:  model.NewSpanID(0),
		Tags:    tags,
		Process: model.NewProcess(testService, tags),
	}
	writer, _ := NewLogzioSpanWriter(LogzioConfig{AccountToken: testAccountToken,
		CustomListenerURL: server.URL,
		Compress:          false,
		InMemoryQueue:     true}, logger)
	assert.NoError(tester, writer.WriteSpan(context.Background(), span))

	time.Sleep(time.Second * 6)
	requests := strings.Split(string(recordedRequests), "\n")
	var logzioSpan objects.LogzioSpan
	assert.NoError(tester, json.Unmarshal([]byte(requests[0]), &logzioSpan))
	assert.Equal(tester, 1, len(logzioSpan.Tag))
	assert.Equal(tester, 1, len(logzioSpan.Process.Tag))

}
