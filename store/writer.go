package store

import (
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/logzio/logzio-go"
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

	err, spanBytes := sw.transformToLogzioSpan(span)

	if err != nil {
		sw.logger.Warn("************************************************************************************", err.Error())
	}
	//sw.logger.Error("refs:" + strconv.Itoa(len(span.GetReferences())))
	//sw.logger.Error("logs:" + strconv.Itoa(len(span.GetLogs())))
	//sw.logger.Error("tags:" + strconv.Itoa(len(span.GetTags())))
	//sw.logger.Error("processTags:" + strconv.Itoa(len(span.GetProcess().Tags)))
	//sw.logger.Error("Sending span: ", spanBytes)

	err = sw.sender.Send(spanBytes)

	if err != nil {
		sw.logger.Warn("************************************************************************************", err.Error())
	}
	sw.sender.Drain()
	//jsonSpan := sw.spanConverter.FromDomainEmbedProcess(span).
	//err = sw.sender.Send(jsonSpan)
	return err
}

func (sw *logzioSpanWriter) transformToLogzioSpan(span *model.Span) (error, []byte) {

	//spanBytes, err := json.Marshal(span)
	//spanString := string(spanBytes)
	var err error
	sw.logger.Error("logzRefs:" + strconv.Itoa(len(sw.transformToLogzioRefs(span.GetReferences()))))
	spanString := sw.spanConverter.FromDomainEmbedProcess(span)
	spanBytes, err := json.Marshal(spanString)
	if err != nil {
		sw.logger.Error(err.Error())
	}
	//sw.logger.Error(spanString)
	m := make(map[string]interface{})
	err = json.Unmarshal(spanBytes,&m)
	if err != nil {
		sw.logger.Error(err.Error())
	}
	m["type"] = "JaegerSpan"
	m["JaegerTags"] = sw.transformToLogzioTags(span.Tags)
	delete(m, "tags")
	delete(m,"process")
	m["process.tags"] = sw.transformToLogzioTags(span.Process.Tags)
	m["process.serviceName"] = span.Process.ServiceName
	m["@timestamp"] = m["startTimeMillis"]

	spanBytes, err = json.Marshal(m)
	return err, spanBytes
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
		logzRef := logzioReference{
			SpanId:  ref.SpanID.String(),
			TraceId: ref.TraceID.String(),
			RefType: ref.RefType.String(),
		}
		result = append(result, logzRef)
	}
	return result
}

func NewLogzioSpanWriter(accountToken string, url string, logger hclog.Logger) *logzioSpanWriter {
	var err error
	var sender *logzio.LogzioSender
	sender, err = logzio.New(
		accountToken,
		logzio.SetUrl("https://" + url + ":8071"),
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