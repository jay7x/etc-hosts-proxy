# This Dockerfile is used by goreleaser

FROM scratch

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/etc-hosts-proxy /bin/etc-hosts-proxy

ENV \
	ETC_HOSTS_PROXY_DEBUG="false" \
	ETC_HOSTS_PROXY_LOG_LEVEL="info" \
	ETC_HOSTS_PROXY_MODE="http" \
	ETC_HOSTS_PROXY_LISTEN_ADDRESS="0.0.0.0:8080"

ENTRYPOINT ["/bin/etc-hosts-proxy"]
CMD ["run"]

EXPOSE 8080
