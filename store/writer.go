package store

import (
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/model/converter/thrift/jaeger"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/logzio/logzio-go"
	"github.com/tidwall/sjson"
	"strconv"
)

type logzioSpanWriter struct {
	accountToken string
	logger   hclog.Logger
	sender   *logzio.LogzioSender
	spanConverter    dbmodel.FromDomain
}

type logzioReference struct {
	SpanId  string `json:"span_id"`
	TraceId string `json:"trace_id"`
	RefType string `json:"ref_type"`
}

type loggerWriter struct {
	logger   hclog.Logger
}

func (lw *loggerWriter) Write(p []byte) (n int, err error) {
	lw.logger.Error(string(p))
	return len(p), nil
}

func (sw *logzioSpanWriter) WriteSpan(span *model.Span) error {
	//msg, err := json.Marshal(map[string]string{"msg":"did you just shit your pants?"})
	var err error
	if sw.sender == nil {
		sw.logger.Error("SSSSSSSSSSSSSSSSSSSSSSender is null")

	}

	err, spanString := sw.transformToLogzioSpan(span)

	if err != nil {
		sw.logger.Warn("************************************************************************************", err.Error())
	}
	sw.logger.Error("refs:" + strconv.Itoa(len(span.GetReferences())))
	sw.logger.Error("logs:" + strconv.Itoa(len(span.GetLogs())))
	sw.logger.Error("tags:" + strconv.Itoa(len(span.GetTags())))
	sw.logger.Error("processTags:" + strconv.Itoa(len(span.GetProcess().Tags)))
	sw.logger.Error("Sending span: ", spanString)

	err = sw.sender.Send([]byte(spanString))

	if err != nil {
		sw.logger.Warn("************************************************************************************", err.Error())
	}
	sw.sender.Drain()
	//jsonSpan := sw.spanConverter.FromDomainEmbedProcess(span).
	//err = sw.sender.Send(jsonSpan)
	return err
}

func (sw *logzioSpanWriter) transformToLogzioSpan(span *model.Span) (error, string) {

	//spanBytes, err := json.Marshal(span)
	//spanString := string(spanBytes)

	sw.logger.Error("logzRefs:" + strconv.Itoa(len(sw.transformToLogzioRefs(span.GetReferences()))))
	spanString := jaeger.FromDomainSpan(span).String()
	spanString, err := sjson.Set(spanString, "type", "jaegerSpan")
	spanString, err = sjson.Set(spanString, "JaegerTags", sw.transformToLogzioTags(span.Tags))
	spanString, err = sjson.Delete(spanString, "tags")
	spanString, err = sjson.Set(spanString, "process.tags", sw.transformToLogzioTags(span.Process.Tags))
	spanString, err = sjson.Set(spanString, "references", sw.transformToLogzioRefs(span.GetReferences()))
	spanString, err = sjson.Set(spanString, "logs", span.GetLogs())
	spanString, err = sjson.Set(spanString, "span_id", span.SpanID.String())
	spanString, err = sjson.Set(spanString, "trace_id", span.TraceID.String())

	return err, spanString
}

func (sw *logzioSpanWriter) transformToLogzioTags(tags []model.KeyValue) map[string]interface{} {
	result := make(map[string]interface{})
	for _, tag := range tags {
		result[tag.Key + ".value"] = tag.Value()
		if tag.GetVType() != model.ValueType_STRING {
			result[tag.Key + ".type"] = tag.GetVType().String()
		}
	}
	return result
}

func (sw *logzioSpanWriter) transformToLogzioRefs(references []model.SpanRef) []logzioReference {
	var result []logzioReference
	for _, ref := range references {
		//result[strconv.Itoa(i) + ".spanID"] = references[i].SpanID.String()
		//result[strconv.Itoa(i) + ".traceID"] = references[i].TraceID.String()
		//result[strconv.Itoa(i) + ".RefType"] = references[i].GetRefType().String()
		logzRef := logzioReference{
			SpanId:  ref.SpanID.String(),
			TraceId: ref.TraceID.String(),
			RefType: ref.RefType.String(),
		}
		result = append(result, logzRef)
	}
	return result
}

func NewLogzioSpanWriter(accountToken string, logger hclog.Logger) *logzioSpanWriter {
	var err error
	var sender *logzio.LogzioSender
	sender, err = logzio.New(
		accountToken,
		logzio.SetUrl("https://listener.logz.io:8071"),
		logzio.SetDebug(&loggerWriter{logger: logger}),
		logzio.SetDrainDiskThreshold(99))

	if err != nil {
		logger.Warn(err.Error(), "********************************************************************")
	}
	w := &logzioSpanWriter{
		accountToken:	accountToken,
		logger: logger,
		sender: sender,
	}

	logger.Warn("Creating new span writer *******************************************")

	msg, err := json.Marshal(map[string]string{"msg":"this is a sample message"})

	logger.Warn(w.accountToken, "yes yews" )
	err = w.sender.Send(msg)
	if err != nil {
		w.logger.Warn("************************************************************************************", err.Error())
	}
	return w
}