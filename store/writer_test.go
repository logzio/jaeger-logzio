package store

import (
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"io/ioutil"
	"jaeger-logzio/store/objects"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testOperation = "testOperation"
	testService = "testService"
)

func TestWriteSpan(tester *testing.T) {
	var recorderRequests []byte
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recorderRequests, _ = ioutil.ReadAll(req.Body)
		//response = append(response, []byte("\n")...)
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

	writer, _ := NewLogzioSpanWriter(LogzioConfig{AccountToken:testAccountToken, CustomListenerURL:server.URL}, logger)
	err := writer.WriteSpan(span)
	if err != nil {
		tester.Errorf("failed write span test: %s", err.Error())
	}
	time.Sleep(time.Second*6)
	requests := strings.Split(string(recorderRequests), "\n")
	var logzioSpan objects.LogzioSpan
	if err := json.Unmarshal([]byte(requests[0]), &logzioSpan); err != nil {
		tester.Errorf("failed to parse recorded request to logzio span: %s", err.Error())
	}
	if logzioSpan.OperationName != testOperation || logzioSpan.Process.ServiceName != testService {
		tester.Errorf("wrong span! got %s", requests[0])
	}

	var logzioService objects.LogzioService
	if err := json.Unmarshal([]byte(requests[1]), &logzioService); err != nil {
		tester.Errorf("failed to parse recorded request to logzio span: %s", err.Error())
	}
	if logzioService.OperationName != testOperation || logzioService.ServiceName != testService {
		tester.Errorf("wrong span! got %s", requests[1])
	}
}

