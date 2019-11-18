package store

import (
	"encoding/json"
	"github.com/jaegertracing/jaeger/model"
	"io/ioutil"
	"testing"
)

func TestTransformToLogzioSpanBytes(tester *testing.T) {
	inStr, err := ioutil.ReadFile("fixtures/domain_01.json")
	if err != nil {
		panic("error opening sample span file")
	}

	var span model.Span
	json.Unmarshal(inStr, &span)
	logzioSpan, err := TransformToLogzioSpanBytes(&span)
	m := make(map[string]interface{})
	err = json.Unmarshal(logzioSpan, &m)
	if _, ok := m["JaegerTag"]; !ok {
		tester.Error("error convetring span to logzioSpan, JaegerTag is not found")
	}
}