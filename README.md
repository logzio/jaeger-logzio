# jaeger-logzio
A storage integration for Jaeger
Build:
```
go build
```
Run:
```
SPAN_STORAGE_TYPE=grpc-plugin jager-all-in-one --grpc-storage-plugin.binary ~/logzio/jaeger-logzio/jaeger-logzio  --grpc-storage-plugin.configuration-file ~/logzio/jaeger-logzio/config.yaml
```
