# Jaeger-Logz.io

Jaeger-Logz.io is a storage option for Jaeger.
It allows Jaeger to store distributed traces on your Logz.io account.

## ⚠️ Warning
We do not reccomend to use this library as standalone integration for traces, if you want to send traces to logz.io we reccomand using the [opentelemetry collector contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib) project with [logzio exprter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/logzioexporter)

**Note**:
This integration requires Logz.io API access.
The Logz.io API is available for Pro and Enterprise accounts.

### Limitations

When you use the Jaeger UI to find traces stored in Logz.io,
there are a couple limitations.
For most users, these won't be an issue,
but they're still good to know:

* **Lookback** must be 48 hours or less
* **Limit Results** must be 4000 traces or less

<!-- tabContainer:start -->
<div class="branching-container">

* [Deploy a single container](#single-container-config)
* [Deploy separate containers](#separate-containers-config)

<!-- tab:start -->
<div id="single-container-config">

These instructions cover deployment
of all the necessary pieces
(Jaeger agent, collector, and query service)
in a single, all-in-one Docker container.

#### Deploy everything in a single container

<div class="tasklist">

##### 1. Create a Docker network

```shell
docker network create net-logzio
```

##### 2. Run the container

You can configure the Logz.io extension with shell variables or environment variables.

For a complete list of options, see the parameters below the code block. 👇

```shell
docker run -d \
 -e ACCOUNT_TOKEN=<<SHIPPING-TOKEN>> \
 -e API_TOKEN=<<API-TOKEN>> \
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

###### Environment variables

| Parameter | Description |
|---|---|
| ACCOUNT_TOKEN (Required) | Required when using as a collector to ship traces to Logz.io. <br> Replace `<<SHIPPING-TOKEN>>` with the [token](https://app.logz.io/#/dashboard/settings/general) of the account you want to ship to. |
| API_TOKEN	(Required) | Required to read back traces from Logz.io. <br> Replace `<<API-TOKEN>>` with an [API token](https://app.logz.io/#/dashboard/settings/api-tokens) from the account you want to use. |
| REGION | Two-letter region code, or blank for US East (Northern Virginia). This determnies your listener URL (where you're shipping the logs to) and API URL. <br> You can find your region code in the [Regions and URLs](https://docs.logz.io/user-guide/accounts/account-region.html#regions-and-urls) table. |
| GRPC_STORAGE_PLUGIN_LOG_LEVEL	(Default: `warn`) | The lowest log level to send. From lowest to highest, log levels are `trace`, `debug`, `info`, `warn`, `error`. <br> Controls logging for Jaeger Logz.io Collector only (not Jaeger components). |

##### 3. Check Jaeger for your traces

Give your traces some time to get from your system to ours, and then open your Jaeger UI.

</div>

</div>
<!-- tab:end -->

<!-- tab:start -->
<div id="separate-containers-config">

These instructions cover deployment
of all the necessary pieces
(Jaeger agent, collector, and query service)
in separate containers.

#### Deploy Jaeger components in separate containers

<div class="tasklist">

##### 1. Create a Docker network

```shell
docker network create net-logzio
```

##### 2. Run Jaeger Logz.io Collector

You can configure the Logz.io extension with shell variables or environment variables.

For a complete list of options, see the parameters below the code block. 👇

```shell
docker run -e ACCOUNT_TOKEN=<<SHIPPING-TOKEN>> \
 --network=net-logzio \
 --name=jaeger-logzio-collector \
 -p 14268:14268 \
 -p 9411:9411 \
 -p 14269:14269 \
 -p 14250:14250 \
logzio/jaeger-logzio-collector:latest
```

###### Environment variables

| Parameter | Description |
|---|---|
| ACCOUNT_TOKEN (Required) | Required when using as a collector to ship traces to Logz.io. <br> Replace `<<SHIPPING-TOKEN>>` with the [token](https://app.logz.io/#/dashboard/settings/general) of the account you want to ship to. |
| REGION | Two-letter region code, or blank for US East (Northern Virginia). This determnies your listener URL (where you're shipping the logs to). <br> You can find your region code in the [Regions and URLs](https://docs.logz.io/user-guide/accounts/account-region.html#regions-and-urls) table. |
| GRPC_STORAGE_PLUGIN_LOG_LEVEL	(Default: `warn`) | The lowest log level to send. From lowest to highest, log levels are `trace`, `debug`, `info`, `warn`, `error`. <br> Controls logging for Jaeger Logz.io Collector only (not Jaeger components). |


###### Ports description
| Port Number | Description |
|---|---|
| 14268 | HTTP protocol. The collector can accept spans directly from clients in jaeger.thrift format over binary thrift protocol |
| 9411 | HTTP protocol. The collector can accept Zipkin spans in Thrift, JSON and Proto (disabled by default) |
| 14269 | HTTP protocol. This is the admin port: health check at `/` and metrics at `/metrics` |
| 14250 | gRPC protocol. This port is used by jaeger-agent to send spans in model.proto format |

##### 3. Run Jaeger query

```shell
docker run --rm -e API_TOKEN=<<API-TOKEN>> \
 --network=net-logzio \
 -p 16686:16686 \
 -p 16687:16687 \
 --name=jaeger-logzio-query \
logzio/jaeger-logzio-query:latest
```

###### Environment variables

| Parameter | Description |
|---|---|
| API_TOKEN	(Required) | Required to read back traces from Logz.io. <br> Replace `<<API-TOKEN>>` with an [API token](https://app.logz.io/#/dashboard/settings/api-tokens) from the account you want to use. |
| REGION | Two-letter region code, or blank for US East (Northern Virginia). This determnies your API URL. <br> You can find your region code in the [Regions and URLs](https://docs.logz.io/user-guide/accounts/account-region.html#regions-and-urls) table. |
| GRPC_STORAGE_PLUGIN_LOG_LEVEL	(Default: `warn`) | The lowest log level to send. From lowest to highest, log levels are `trace`, `debug`, `info`, `warn`, `error`. <br> Controls logging for Jaeger Logz.io Collector only (not Jaeger components). |

##### 4. Run Jaeger agent

You can run your own instance of Jaeger agent.
If you're not already running Jaeger agent,
start it up with this command:

```shell
docker run --rm --name=jaeger-agent --network=net-logzio \
 -p5775:5775/udp \
 -p6831:6831/udp \
 -p6832:6832/udp \
 -p5778:5778/tcp \
 jaegertracing/jaeger-agent:1.18.0 \
 --reporter.grpc.host-port=jaeger-logzio-collector:14250
```

##### 5. Check Jaeger for your traces

Give your traces some time to get from your system to ours, and then open your Jaeger UI.

</div>

</div>
<!-- tab:end -->

</div>
<!-- tabContainer:end -->


----

## Customizing shipping & API URLs

When you use the `REGION` environment variables, your listener URL and API URL are automatically set.
For dev & testing purposes,
you can override region settings by using these environment variables.

| Parameter | Description |
|---|---|
| CUSTOM_LISTENER_URL	| Set a custom URL to ship logs to (e.g., `http://localhost:9200`). This overrides the `REGION` environment variable. |
| CUSTOM_API | Set a custom API URL (e.g., `http://localhost:9200/_msearch`). This overrides the `REGION` environment variable. |

## Customizing storage

By default, the queue is saved on disk You can also specify a custom directory to store the queue in

| Parameter | Description | Default value |
|---|---|---|
| CUSTOM_QUEUE_DIR| Path to a directory you want to store the queue in | none |
| DRAIN_INTERVAL| Queue drain interval in seconds | `3` |


You can configure Jaeger-Logz.io the save the queue in memory and set log count limit and queue capacity:

| Parameter | Description | Default value |
|---|---|---|
| IN_MEMORY_QUEUE| If the parameter is set to `true`, the queue wil be saved in memory, and override any disk queue configuration| `false` |
| IN_MEMORY_CAPACITY| In memory queue capacity in bytes | `20 * 1024 * 1024` 20mb |
| LOG_COUNT_LIMIT| Max number of items allowed in the queue, **note** this parameter is not relevant for disk queue| `500000` |
| DRAIN_INTERVAL| Queue drain interval in seconds | `3` |


## Data compression
All bulks are compressed with gzip by default, to disable compressing initialize `COMPRESS` env variable set to `false`

## Run go binary with bash

Clone this repo and change `config.yaml` to fit your Logz.io account parameters.
Example:
```yaml
region: "us"
apiToken: "api-token"
accountToken: "sapmle-token"
drainInterval: 5
customListenerUrl: "http://custom.com"
compress: true
inMemoryQueue: true
inMemoryCapacity: 20 * 1024 * 1024
logCountLimit: 10000
```
Then, build Logz.io binary:

```
go build
```

#### Clone and build jaeger all in one binary:

Follow the [Getting Started](https://github.com/jaegertracing/jaeger/blob/master/CONTRIBUTING.md#getting-started) from the Jaeger's repo.
Build the Jaeger all-in-one binary or download it from the [Jaeger releases page](https://github.com/jaegertracing/jaeger/releases):

**NOTE**: If you intend to run the generated binary file from the build on a unix base system, set this env variable first:`export GOOS=linux`

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
**Note**: if you're not running the all-in-one container, you should replace "jaeger-logzio" with the name of the host/container which runs the collector.

Then navigate to http://localhost:8080 .

### Changelog
- v1.0.6
  - Update logzio-go (v1.0.5 -> v1.0.6)

- v1.0.5
  - Update logzio-go (v1.0.3 -> v1.0.5)
  - Fix region code bug

- v1.0.4
  - Logzio-go -> 1.0.3

- v1.0.3
  - Support for in memory queue
  - Added gzip compression

- v1.0.2
  - Changed the default queue directory from ~/tmp -> $HOME/tmp
  - Changed custom queue directory validation to be windows compatible
   
- v1.0.1
   - Support for custom queue directory

- v1.0.0 - **Breaking Changes**
   - Support for searching traces by tags is affected by the introduction of new tags.
   - Static image versions of Jaeger components (1.18)
   - Fix empty tags bug
       - Since the deprecation of TChannel in version 1.16, it is necessary to use gRPC reporter protocol when running a standalone Jaeger agent. - see "Run Jaeger agent"
- v0.0.3
   - Fix x509 certificate issue 
