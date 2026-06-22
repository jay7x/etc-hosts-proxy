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

## Environment / CLI

CLI is `etc-hosts-proxy run` (`-H`/`--hosts`, `-M`/`--mode`, `-L`/`--listen-address`). Every flag also has an `ETC_HOSTS_PROXY_*` env-var counterpart. Default mode is `http`; default listen `127.0.0.1:8080`.
