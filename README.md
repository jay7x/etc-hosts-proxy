# etc-hosts-proxy

> Stop changing `/etc/hosts` file! Use `etc-hosts-proxy` instead!

When testing a website there are usually 3 ways to access the test environment while keeping the `Host` HTTP header:

1. Override the hostname in the `/etc/hosts` file. Admin privileges are required for this.

2. Use local DNS server with ability to override the hostname (`dnsmasq` e.g.). You should configure the local DNS server properly. Then you should change your network settings to use it.

3. Use a proxy server which can override the hostname in a request.

This proxy implements the 3rd way. Nor admin privileges neither any system-wide changes are required for this. Just start the proxy server and direct your browser to use it.

A proxy-switching extension for your browser (FoxyProxy or SwitchyOmega e.g.) is highly recommended.

## What this proxy does and what's not

- :white_check_mark: HTTP proxy protocol is supported
- :white_check_mark: HTTPS CONNECT method is supported
- :green_square: SOCKS5 support is expected
- :green_square: HTTP proxy-chaining support is expected
- :x: Config file support is possible but not planned
- :x: Daemon mode support is possible but not planned
- :x: HTTPS MitM support is not planned (use mitmproxy)
- :x: Request/response rewrite support not planned (use mitmproxy)
- :x: ACL support is not planned

## Usage

Run the HTTP proxy on 127.0.0.1:8080 and redirect some hostnames to a local web server:

```bash
etc-hosts-proxy run -H example.com=127.0.0.1 -H www.example.com=127.0.0.1
```

Note: you may use comma-separated list of `<host>=<ip>` pairs in a single `-H` option too: `-H example.com=127.0.0.1,www.example.com=127.0.0.1`

Test the above with curl:

```bash
curl -v -x 127.0.0.1:8080 http://example.com
curl -v -x 127.0.0.1:8080 http://www.example.com
```

Proxy listens on `127.0.0.1:8080` by default. Use `-L` (or `--listen-address`) CLI option to change this.

## Docker usage

You may use a docker image too if you'd like:

```bash
# Run the proxy in background
docker run -d --name=etc-hosts-proxy -p 8080:8080 --rm \
  -e ETC_HOSTS_PROXY_HOSTS_LIST="akamai.com=2.21.250.7,www.akamai.com=2.21.250.7" \
  -e ETC_HOSTS_PROXY_DEBUG=true \
  ghcr.io/jay7x/etc-hosts-proxy:latest

curl -v -x 127.0.0.1:8080 http://akamai.com

docker logs etc-hosts-proxy
```

NOTE: You should not use 127.0.0.1 (or ::1) as your redirection target in the hosts list while running in a container. This will redirect the request to the container's localhost, which is not what you might expect.

A bit more complex example to redirect some domains to a nginx container:

```bash
# Create a docker network
docker network create somenet

# Run a web server exposed in somenet and on 0.0.0.0:8080 on the host
docker run -d --name=nginx --net=somenet -p 8080:80 --rm nginx:latest

# Run the proxy connected to somenet and exposed on 0.0.0.0:3128 on the host
docker run -d --name=etc-hosts-proxy --net=somenet -p 3128:8080 --rm \
  -e ETC_HOSTS_PROXY_HOSTS_LIST="example.com=nginx,www.example.com=nginx" \
  -e ETC_HOSTS_PROXY_DEBUG=true \
  ghcr.io/jay7x/etc-hosts-proxy:latest

# Check nginx
curl -v http://127.0.0.1:8080

# Check proxy is proxying (note the port is 3128 here)
curl -v -x 127.0.0.1:3128 http://example.com
curl -v -x 127.0.0.1:3128 http://www.example.com
# This should not be redirected
curl -v -x 127.0.0.1:3128 http://example.net

# Check logs
docker logs proxy
docker logs nginx

# Cleanup
docker stop proxy
docker stop nginx
docker network rm somenet
```

Docker images are built by `goreleaser` from the [Dockerfile](https://github.com/jay7x/etc-hosts-proxy/blob/main/.dockerfile)

See [etc-hosts-proxy Github Container registry](https://github.com/jay7x/etc-hosts-proxy/pkgs/container/etc-hosts-proxy) for more details

## Environment variables

| Variable | Description |
| - | - |
| `ETC_HOSTS_PROXY_DEBUG` | Enable debug mode |
| `ETC_HOSTS_PROXY_LOG_LEVEL` | Set the logging level [`trace`, `debug`, `info`, `warn`, `error`] |
| `ETC_HOSTS_PROXY_MODE` | Mode to start proxy in (`http` or `socks5`) |
| `ETC_HOSTS_PROXY_LISTEN_ADDRESS` | [`<host>`]:`<port>` to listen for proxy requests on |
| `ETC_HOSTS_PROXY_HOSTS_LIST` | comma-separated list of `<host>=<ip>` pairs to resolve `<host>` to `<ip>` |
