package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"jaeger-logzio/store/objects"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger/storage/spanstore"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/olivere/elastic"
)

var (
	logger = hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Name:       "jaeger-logzio-tests",
		JSONFormat: true,
	})
	recordedRequests []byte
	server           = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recordedRequests, _ = ioutil.ReadAll(req.Body)
		rw.WriteHeader(http.StatusOK)
		if strings.Contains(string(recordedRequests), "{\"aggregations\":{\"traceIDs\"") {
			resp, _ := ioutil.ReadFile("fixtures/trace_ids_response.json")
			_, _ = rw.Write(resp)
		}
	}))
	reader = NewLogzioSpanReader(LogzioConfig{APIToken: testAPIToken, CustomAPIURL: server.URL}, logger)
)

func checkRecordedRequestAndGetBody(tester *testing.T, requestCount int) string {
	requestLines := strings.Split(string(recordedRequests), "\n")
	if len(requestLines) != (requestCount*2)+1 {
		tester.Fatalf("wrong number of requests. expected %d got : %d", (requestCount*2)+1, len(requestLines)) // n * (header + body) + empty NewLine
	}
	fullBody := ""
	for i := 1; i < len(requestLines); i += 2 {
		reqBody := requestLines[i]
		var searchRequest elastic.SearchRequest
		if err := json.Unmarshal([]byte(reqBody), &searchRequest); err != nil {
			tester.Fatalf("request is not in Elasticsearch format, can't parse: %s", err.Error())
		}
		fullBody = fullBody + requestLines[i]
	}
	return fullBody
}

func TestGetTrace(tester *testing.T) {
	_, _ = reader.GetTrace(context.Background(), model.TraceID{Low: 1, High: 0})
	reqBody := checkRecordedRequestAndGetBody(tester, 1)
	assert.True(tester, strings.Contains(reqBody, "\"traceID\":\"1\""), "trace id incorrect or not exist")
}

func TestGetServices(tester *testing.T) {
	_, _ = reader.GetServices(context.Background())
	reqBody := checkRecordedRequestAndGetBody(tester, 1)
	assert.True(tester, strings.Contains(reqBody, "\"field\":\"serviceName\""), "serviceName field is not in request")
}

func TestGetOperations(tester *testing.T) {
	_, _ = reader.GetOperations(context.Background(), testService)
	reqBody := checkRecordedRequestAndGetBody(tester, 1)
	assert.True(tester, strings.Contains(reqBody, "\"field\":\"operationName\""), "operationName field is not in request")
	assert.True(tester, strings.Contains(reqBody, fmt.Sprintf("{\"term\":{\"serviceName\":\"%s\"}}", testService)), "service filter is incorrect or not exist")
}

func TestFindTraces(tester *testing.T) {
	const minTime = 1000000
	const maxTime = 2000000
	query := spanstore.TraceQueryParameters{
		ServiceName:   testService,
		OperationName: testOperation,
		StartTimeMin:  time.Unix(0, minTime*1000),
		StartTimeMax:  time.Unix(0, maxTime*1000),
	}
	_, _ = reader.FindTraces(context.Background(), &query)
	reqBody := checkRecordedRequestAndGetBody(tester, 2)

	assert.True(tester,
		strings.Contains(
			reqBody,
			fmt.Sprintf("{\"range\":{\"startTime\":{\"from\":%d,\"include_lower\":true,\"include_upper\":true,\"to\":%d}}}", minTime, maxTime)),
		"request time range is incorrect or not exist")

	assert.True(tester, strings.Contains(reqBody, "{\"term\":{\"traceID\":\"42\"}"), "missing traceID term for trace id 42")
	assert.True(tester, strings.Contains(reqBody, "{\"term\":{\"traceID\":\"314\"}"), "missing traceID term for trace id 314")
}

func TestFindTraceIDs(tester *testing.T) {
	const minTime = 1000000
	const maxTime = 2000000
	tags := map[string]string {
		"name":			testName,
		"example.Key":	testValue,
	}
	query := spanstore.TraceQueryParameters{
		ServiceName:   testService,
		OperationName: testOperation,
		StartTimeMin:  time.Unix(0, minTime*1000),
		StartTimeMax:  time.Unix(0, maxTime*1000),
		Tags:		   tags,
	}
	_, _ = reader.FindTraceIDs(context.Background(), &query)

	reqBody := checkRecordedRequestAndGetBody(tester, 1)
	assert.True(tester,
		strings.Contains(
			reqBody,
			fmt.Sprintf("{\"range\":{\"startTime\":{\"from\":%d,\"include_lower\":true,\"include_upper\":true,\"to\":%d}}}", minTime, maxTime)),
		"request time range is incorrect or not exist")

	assert.True(tester, strings.Contains(reqBody,
		fmt.Sprintf("{\"match\":{\"process.serviceName\":{\"query\":\"%s\"}}}", testService)),
		"service filter is incorrect or not exist")

	assert.True(tester,
		strings.Contains(reqBody, fmt.Sprintf("{\"match\":{\"operationName\":{\"query\":\"%s\"}}}", testOperation)),
		"operation filter is incorrect or not exist")

	for key, value := range tags {
		assert.True(tester,
			strings.Contains(reqBody,fmt.Sprintf("{\"must\":{\"match\":{\"JaegerTag.%s\":{\"query\":\"%s\"}}}}",
				strings.ReplaceAll(key,".", objects.TagDotReplacementCharacter) ,value)),
			fmt.Sprintf("tag filter for '%s' is incorrect or not exist", key))

		assert.True(tester,
			strings.Contains(reqBody,fmt.Sprintf("{\"must\":{\"match\":{\"process.tag.%s\":{\"query\":\"%s\"}}}}",
				strings.ReplaceAll(key,".", objects.TagDotReplacementCharacter) ,value)),
			fmt.Sprintf("tag filter for '%s' is incorrect or not exist", key))
	}
}