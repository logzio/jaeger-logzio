package store

import (
	"encoding/json"
	"io/ioutil"
	"jaeger-logzio/store/objects"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
)

const (
	testOperation = "testOperation"
	testService   = "testService"
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

	writer, _ := NewLogzioSpanWriter(LogzioConfig{AccountToken: testAccountToken, CustomListenerURL: server.URL}, logger)
	assert.NoError(tester, writer.WriteSpan(span))

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
