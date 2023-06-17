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
