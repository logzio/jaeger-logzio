# jaeger-logzio

This is the repository that contains Logz.io Storage gRPC plugin for Jaeger.

### Limitations

When you use the Jaeger UI to find traces stored in Logz.io, there are a couple limitations.
For most users, these won't be an issue, but they're still good to know:

* **Lookback** must be 48 hours or less
* **Limit Results** must be 1000 traces or less

## Run Jaeger with Logz.io storage in a docker

First create a docker network:
```
docker network create net-logzio
```
If you want to run jaeger-logzio all-in-one (logz.io storage, Jaeger agent, collector and query) the command below 👇 with the following environment variables:

| Parameter | Description |
|---|---|
| ACCOUNT_TOKEN | **Required**.<br> Required when using as a collector to ship traces to Logz.io. <br> Replace `<ACCOUNT-TOKEN>` with the [token](https://app.logz.io/#/dashboard/settings/general) of the account you want to ship to. |
| API_TOKEN | **Required**.<br> Required to read back traces from Logz.io. <br> Replace `<API-TOKEN>` with the [API token](https://app.logz.io/#/dashboard/settings/api-tokens) from the account you want to use. |
| REGION | **Default**: `us` <br> Two-letters region code. Replace `us` with your region's code. For more information on finding your account's region, see [Account region](https://docs.logz.io/user-guide/accounts/account-region.html). |
| CUSTOM_LISTENER_URL | Use this to set a custom listener URL (e.g http://localhost:9200). This config will override the region variable.|
| CUSTOM_API | Use this to set a api URL (e.g http://localhost:9200/_msearch). This config will override the region variable.|

```
docker run -d -e ACCOUNT_TOKEN=<<ACCOUNT_TOKEN>> -e API_TOKEN=<<API_TOKEN>> \
  --name=jaeger-logzio \
  --network=net-logzio \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 9411:9411 \
logzio/jaeger-logzio:latest

```

If you want to run the jaeger-logzio with the Jaeger collector only, use the following commands instead
```
docker run -e ACCOUNT_TOKEN=<<ACCOUNT_TOKEN>> \
  --network=net-logzio \
  --name=jaeger-logzio \
  -p 14268:14268 \
  -p 9411:9411 \
  -p 14267:14267 \
  -p 14269:14269 \
  -p 14250:14250 \
logzio/jaeger-logzio-collector:latest
```

**Note**: Jaeger collector can't run without a Jaeger agent, if you're not sure what it means, just run this command:
```
docker run --rm --name=jaeger-agent --network=net-logzio \
  -p5775:5775/udp \
  -p6831:6831/udp \
  -p6832:6832/udp \
  -p5778:5778/tcp \
  jaegertracing/jaeger-agent:1.9 \
  --reporter.tchannel.host-port=jaeger-logzio:14267
```

## Run go binary with bash

Clone this repo and change config.yaml to fit your Logz.io account parameters.
Then, build Logz.io binary:

```
go build
```

#### Clone and build jaeger all in one binary:

Follow the [Getting Started](https://github.com/jaegertracing/jaeger/blob/master/CONTRIBUTING.md#getting-started) from the Jaeger's repo.
Build the Jaeger all-in-one binary:

```
go build -tags ui
```

Run Jaeger all-in-one binary with Logz.io storage:

```
SPAN_STORAGE_TYPE=grpc-plugin  ./cmd/all-in-one/all-in-one --grpc-storage-plugin.binary ~/path/to/jaeger-logzio/jaeger-logzio  --grpc-storage-plugin.configuration-file ~/path/to/jaeger-logzio/config.yaml
```

## Example
HotROD (Rides on Demand) is a demo application by Jaeger that consists of several microservices and illustrates the use of the OpenTracing API.
It can be run standalone, but requires Jaeger backend to view the traces.
You can try and run to view sample traces:
```
docker run --rm -it \
  -p8080-8083:8080-8083  --network=net-logzio  \
  -e JAEGER_AGENT_HOST="jaeger-logzio" \
  jaegertracing/example-hotrod:1.9 \
  all
```

Then navigate to http://localhost:8080 .