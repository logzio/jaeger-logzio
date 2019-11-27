package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jaegertracing/jaeger/model"
	"github.com/olivere/elastic"
	"github.com/pkg/errors"
	"jaeger-logzio/store/objects"
)

func convertTraceIDsStringsToModels(traceIDs []string) ([]model.TraceID, error) {
	traceIDsModels := make([]model.TraceID, len(traceIDs))
	for i, ID := range traceIDs {
		traceID, err := model.TraceIDFromString(ID)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Making traceID from string '%s' failed", ID))
		}

		traceIDsModels[i] = traceID
	}
	return traceIDsModels, nil
}

func unmarshalJSONSpan(esSpanRaw *elastic.SearchHit) (*objects.LogzioSpan, error) {
	esSpanInByteArray := esSpanRaw.Source

	var jsonSpan objects.LogzioSpan

	decoder := json.NewDecoder(bytes.NewReader(*esSpanInByteArray))
	decoder.UseNumber()
	if err := decoder.Decode(&jsonSpan); err != nil {
		return nil, err
	}
	return &jsonSpan, nil
}

