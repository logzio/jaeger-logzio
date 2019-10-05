# jaeger-logzio
A storage integration for Jaeger

Build:
```
go build
```
Run:
```
git clone https://github.com/jaegertracing/jaeger
cd https://github.com/jaegertracing/jaeger/cmd/all-in-one
git checkout v1.12.0
go build
SPAN_STORAGE_TYPE=grpc-plugin jaeger-all-in-one --grpc-storage-plugin.binary ~/path/to/jaeger-logzio  --grpc-storage-plugin.configuration-file ~/path/to/config.yaml
```
