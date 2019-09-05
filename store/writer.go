package store

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/model/converter/thrift/jaeger"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"github.com/logzio/logzio-go"
	"github.com/tidwall/sjson"
)

type logzioSpanWriter struct {
	accountToken string
	logger   hclog.Logger
	sender   *logzio.LogzioSender
	spanConverter    dbmodel.FromDomain
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

	//sw.logger.Warn("tokne: ", sw.accountToken,": sending span ****************************************************************************************************")
	//err = sw.sender.Send([]byte(msg))
	//if err != nil {
	//	sw.logger.Warn("************************************************************************************", err.Error())
	//}
	//var spanBytes []byte
	//jsonSpan := sw.spanConverter.FromDomainEmbedProcess(span)

	//sw.logger.Error("SSSSSSSSSSSSSSSSSSSSSSPPPPPPPPAAAAANNNN: ", span.)
	err, spanString := sw.transformToLogzioSpan(span)

	if err != nil {
		sw.logger.Warn("************************************************************************************", err.Error())
	}
	sw.logger.Error("logs:", len(span.GetLogs()))
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

	spanString := jaeger.FromDomainSpan(span).String()
	spanString, err := sjson.Set(spanString, "type", "jaegerSpan")
	spanString, err = sjson.Set(spanString, "SchwaegerTags", sw.transformToLogzioTags(span.Tags))
	spanString, err = sjson.Set(spanString, "process.tags", sw.transformToLogzioTags(span.Process.Tags))
	spanString, err = sjson.Delete(spanString, "tags")
	spanString, err = sjson.Set(spanString, "span_id", fmt.Sprintf("%x", int(span.SpanID)))
	spanString, err = sjson.Set(spanString, "trace_id", fmt.Sprintf("%x", span.TraceID.Low))

	return err, spanString
}

func (sw *logzioSpanWriter) transformToLogzioTags(tags []model.KeyValue) map[string]interface{} {
	result := make(map[string]interface{})
	for i:=0 ; i<len(tags) ; i++  {
			result[tags[i].Key + ".value"] = tags[i].Value()
		result[tags[i].Key + ".type"] = tags[i].GetVType().String()
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
	//w.sender, err = logzio.New(
	//	"oCwtQDtWjDOMcHXHGGNrnRgkEMxCDuiO",
	//	logzio.SetUrl("https://listener.logz.io:8071"))
	//
	//if err != nil {
	//	logger.Warn(err.Error(), "********************************************************************")
	//}
	msg, err := json.Marshal(map[string]string{"msg":"this is a sample message"})

	logger.Warn(w.accountToken, "yes yews" )
	err = w.sender.Send(msg)
	if err != nil {
		w.logger.Warn("************************************************************************************", err.Error())
	}
	return w
}