FROM jaegertracing/all-in-one:1.13

ENV SPAN_STORAGE_TYPE grpc-plugin
ENV GRPC_STORAGE_PLUGIN_BINARY "/go/bin/jaeger-logzio"

#COPY ./tmp /tmp

COPY ./jaeger-logzio /go/bin/

