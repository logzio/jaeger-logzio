docker run -d -e ACCOUNTTOKEN=<<ACCOUNT_TOKEN>> -e LISTENERURL=https://listener.logz.io:8071 -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 9411:9411 \
logzio/jaeger-logzio:latest
