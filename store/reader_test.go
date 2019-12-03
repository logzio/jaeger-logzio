package store

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/olivere/elastic"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	logger = hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
 	recordedRequests []byte
	server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recordedRequests, _ = ioutil.ReadAll(req.Body)
		rw.WriteHeader(http.StatusOK)
		logger.Info(string(recordedRequests))
	}))
	reader = NewLogzioSpanReader(LogzioConfig{APIToken:testAPIToken, CustomAPIURL: server.URL}, logger)
)


func TestGetTrace(tester *testing.T) {
	_, _ = reader.GetTrace(context.Background(), model.TraceID{Low:1, High:0})
	requestLines := strings.Split(string(recordedRequests),"\n")
	if len(requestLines) != 3 {
		tester.Fatalf("wrong number of requests. expected 3 got : %d", len(recordedRequests)) // request + header + empty NewLine = 3
	}
	var boolQuery elastic.SearchRequest
	if err := json.Unmarshal([]byte(requestLines[1]), &boolQuery); err != nil {
		tester.Errorf("can't parse request to json: %s", err.Error())
	}

	if !strings.Contains(requestLines[1],"\"traceID\":\"1\"") {
		tester.Errorf("trace id incorrect or not exist")
	}
}