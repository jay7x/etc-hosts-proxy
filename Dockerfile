FROM golang:latest AS BUILD

COPY *.go /go/src/
COPY go.* /go/src/

RUN cd /go/src && go build

FROM debian:stable-slim

COPY --from=build /go/src/etc-hosts-proxy /usr/bin
RUN useradd -s /bin/bash -m -d /home/etc-hosts-proxy etc-hosts-proxy

COPY entrypoint.sh /entrypoint.sh

USER etc-hosts-proxy

ENV ETC_HOSTS_PROXY_BIND_ADDRESS 0.0.0.0:8080
ENV ETC_HOSTS_PROXY_MODE http
ENV ETC_HOSTS_PROXY_HOSTS_LIST ""

ENTRYPOINT ["/entrypoint.sh"]
CMD ["run"]

EXPOSE 8080
