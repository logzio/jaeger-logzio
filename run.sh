#!/usr/bin/env bash
docker run -d --rm -e ACCOUNT_TOKEN=<<ACCOUNT_TOKEN>> -e API_TOKEN=<<API_TOKEN>> \
  --network=net-logzio \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 9411:9411 \
logzio/jaeger-logzio:latest