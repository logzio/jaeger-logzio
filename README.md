# jaeger-logzio

A storage integration for Jaeger

Clone this repo and change config.yaml to fit your Logz.io account parameters.
Then, create Logz.io binary:

```
go build
```

Build jaeger all in one binary:

```
git clone https://github.com/jaegertracing/jaeger
cd https://github.com/jaegertracing/jaeger/cmd/all-in-one
go build
```

Run jaeger with Logz.io as a storage:

```
SPAN_STORAGE_TYPE=grpc-plugin all-in-one --grpc-storage-plugin.binary ~/path/to/jaeger-logzio  --grpc-storage-plugin.configuration-file ~/path/to/config.yaml
```

[Run HotROD from source](https://github.com/jaegertracing/jaeger/tree/master/examples/hotrod#run-hotrod-from-source) 