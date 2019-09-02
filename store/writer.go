package store

import (
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
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
	spanBytes, err := json.Marshal(span)
	spanString := string(spanBytes)
	spanString, err	= sjson.Set(spanString,"jaegerTags",span.Tags)
	spanString, err = sjson.Delete(spanString,"tags")
	if err != nil {
		sw.logger.Warn("************************************************************************************", err.Error())
	}
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