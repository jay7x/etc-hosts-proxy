# Agents

Single `main`-package Go module at root. No codegen.

## Commands

```bash
go build ./...              # build
go vet ./...                # typecheck
golangci-lint run           # lint (no custom config, uses defaults)
go test -v ./...            # acceptance tests (see below)
goreleaser release --snapshot --clean   # dry-run release
```

## Go version

CI tests against Go 1.25 and 1.26.

## Release

Tags `v*.*.*` trigger goreleaser, which cross-compiles binaries and publishes Docker images to `ghcr.io/jay7x/etc-hosts-proxy`.

## Docker

Goreleaser-owned (see `.goreleaser.yml`). For manual builds, run `go build` first, then use the `Dockerfile`.

## Acceptance tests

Tests are in `acceptance_test.go`, package `main`. They build the binary, spawn it as a subprocess, and assert end-to-end behavior against a real hello server:

| Test | Mode | What it validates |
|---|---|---|
| `TestHTTP_rewritesHost` | HTTP | HTTP GET through proxy arrives at rewritten destination |
| `TestCONNECT_rewritesHost` | HTTP CONNECT | CONNECT tunnel + HTTP GET through tunnel reaches rewritten destination |
| `TestSOCKS5_rewritesHost` | SOCKS5 | SOCKS5 dial + HTTP GET through tunnel reaches rewritten destination |

All tests check on the server side: a `httptest.Server` returns `"hello"`, the test sends an HTTP request through the proxy, and asserts the response body is `"hello"`.

The SOCKS5 test uses `test.localhost` (RFC 6761 — always resolves to loopback) to avoid needing DNS. The `MapResolver` TODO in `TODO.md` covers rewriting to non-loopback addresses in SOCKS5 mode.

The test binary (`./etc-hosts-proxy.test`) is automatically removed after the test run via `TestMain`.

## Access logs

Access logs are emitted as structured `logrus.WithFields` entries at `info` level for every proxied request:

| Mode | Fields |
|---|---|
| **HTTP** | `client`, `method` (GET/POST/...), `host`, `path`, `status`, `size`, `duration_ms`, `rewritten`, `target` |
| **CONNECT** | `client`, `method: "CONNECT"`, `host`, `rewritten`, `target` |
| **SOCKS5** | `client`, `method: "SOCKS5"`, `host`, `rewritten`, `target` |

`target` is omitted when `rewritten=false`. Access logs are suppressed at `warn` level and above; `debug` level adds goproxy/socks5 verbose output.

## Environment / CLI

CLI is `etc-hosts-proxy run` (`-H`/`--hosts`, `-M`/`--mode`, `-L`/`--listen-address`). Every flag also has an `ETC_HOSTS_PROXY_*` env-var counterpart. Default mode is `http`; default listen `127.0.0.1:8080`.

Global flags: `--log-level`, `--debug`, `--log-format` (`text`/`json`, default `text`). When `json` is selected, access logs are emitted as JSON lines and can be parsed with `jq` or similar tools.
