FROM scratch

COPY etc-hosts-proxy /bin/etc-hosts-proxy

ENV ETC_HOSTS_PROXY_LISTEN_ADDRESS 0.0.0.0:8080

ENTRYPOINT ["/bin/etc-hosts-proxy"]
CMD ["run"]

EXPOSE 8080
