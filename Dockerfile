FROM jaegertracing/all-in-one:latest

ENV SPAN_STORAGE_TYPE grpc-plugin
ENV GRPC_STORAGE_PLUGIN_BINARY "/go/bin/jaeger-logzio"

COPY ./jaeger-logzio /go/bin/

