# jaeger-logzio

This is the repository that contains Logz.io Storage gRPC plugin for Jaeger.

### Limitations

When you use the Jaeger UI to find traces stored in Logz.io, there are a couple limitations.
For most users, these won't be an issue, but they're still good to know:

* **Lookback** must be 48 hours or less
* **Limit Results** must be 1000 traces or less

## Run as a docker

Run the command below 👇 with the following environment variables:

| Parameter | Description |
|---|---|
| ACCOUNT_TOKEN | **Required**.<br> Required when using as a collector to ship traces to Logz.io. <br> Replace `<ACCOUNT-TOKEN>` with the [token](https://app.logz.io/#/dashboard/settings/general) of the account you want to ship to. |
| API_TOKEN | **Required**.<br> Required to read back traces from Logz.io. <br> Replace `<API-TOKEN>` with the [API token](https://app.logz.io/#/dashboard/settings/api-tokens) from the account you want to use. |
| REGION | **Default**: `us` <br> Two-letters region code. Replace `us` with your region's code. For more information on finding your account's region, see [Account region](https://docs.logz.io/user-guide/accounts/account-region.html). |

```
docker run -d -e ACCOUNT_TOKEN=<ACCOUNT_TOKEN> -e API_TOKEN=<API-TOKEN> -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 9411:9411 jaeger-logzio
logzio/jaeger-logzio:latest

```

if you want to run the jaeger-logzio with the jaeger collector only, use the following command instead
```
docker run -d -e ACCOUNT_TOKEN=<<ACCOUNT_TOKEN>> -e \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 14268:14268 \
  -p 9411:9411 \
  -p 14267:14267 \
  -p 14250:14250 \
logzio/jaeger-logzio-collector:latest
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
You can try running [HotROD example](https://github.com/jaegertracing/jaeger/tree/master/examples/hotrod#run-hotrod-from-source) to test it out.