# jaeger-logzio

A storage integration for Jaeger

## Run as a docker

Run the command below 👇 with the following parameters:

| Parameter | Description |
|---|---|
| ACCOUNTTOKEN | **Required**. Your Logz.io [account token](https://app.logz.io/#/dashboard/settings/manage-accounts) <br> Replace `<<ACCOUNT-TOKEN>>` with the [token](https://app.logz.io/#/dashboard/settings/general) of the account you want to ship to. |
| LISTENERURL | **Default**: `https://listener.logz.io:8071` <br>  Listener URL and port. Replace `listener.logz.io` with your region's listener host. For more information on finding your account's region, see [Account region](https://docs.logz.io/user-guide/accounts/account-region.html). |

```
docker run -d -e ACCOUNTTOKEN=<<ACCOUNT_TOKEN>> -e LISTENERURL=https://listener.logz.io:8071 -p 5775:5775/udp \
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
docker run -d -e ACCOUNTTOKEN=VhrAoIvDaHcqHRdChvrrALbbAJpJkYpx -e LISTENERURL=https://listener-eu.logz.io:8071 \
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