package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/spf13/cobra"
	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
)

type accessEntry struct {
	start     time.Time
	client    string
	method    string
	host      string
	path      string
	rewritten bool
	target    string
}

func newRunCommand() *cobra.Command {
	var runCommand = &cobra.Command{
		Use:   "run",
		Short: "Run the proxy",
		Args:  cobra.NoArgs,
		RunE:  runAction,
		Example: fmt.Sprintf(`
  Run the HTTP proxy on 127.0.0.1:8080 and redirect some hostnames to a local web server:

  %s run -H example.com=127.0.0.1 -H www.example.com=127.0.0.1
  - or -
  %s run -H example.com=127.0.0.1,www.example.com=127.0.0.1

  Test the above with curl:
  curl -v -x 127.0.0.1:8080 http://example.com
  curl -v -x 127.0.0.1:8080 http://www.example.com
`,
			executableName,
			executableName,
		),
	}

	runCommand.Flags().StringToStringP("hosts", "H",
		GetEnvStrMap("ETC_HOSTS_PROXY_HOSTS_LIST"),
		"<host>=<ip> pairs to redirect <host> to <ip> (ETC_HOSTS_PROXY_HOSTS_LIST)")
	runCommand.Flags().StringP("mode", "M",
		GetEnvWithDefault("ETC_HOSTS_PROXY_MODE", "http"),
		"Mode to start proxy in (http or socks5) (ETC_HOSTS_PROXY_MODE)")
	runCommand.Flags().StringP("listen-address", "L",
		GetEnvWithDefault("ETC_HOSTS_PROXY_LISTEN_ADDRESS", "127.0.0.1:8080"),
		"[<host>]:<port> to listen for proxy requests on (ETC_HOSTS_PROXY_LISTEN_ADDRESS)")
	return runCommand
}

type socks5Logger struct {
	logger *slog.Logger
}

func (l socks5Logger) Errorf(format string, args ...interface{}) {
	l.logger.Log(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
}

// SOCKS5 destination address rewriter
type HostRewriter struct {
	hostsMap map[string]string
}

func (r HostRewriter) Rewrite(ctx context.Context, request *socks5.Request) (context.Context, *statute.AddrSpec) {
	originalHost := request.DestAddr.FQDN
	if originalHost == "" {
		originalHost = request.DestAddr.IP.String()
	}

	dst, found := r.hostsMap[request.DestAddr.FQDN]
	if !found {
		dst, found = r.hostsMap[request.DestAddr.IP.String()]
	}

	if found {
		daSpec, err := statute.ParseAddrSpec(net.JoinHostPort(dst, strconv.Itoa(request.DestAddr.Port)))
		if err == nil {
			slog.Info("",
				slog.String("client", request.RemoteAddr.String()),
				slog.String("method", "SOCKS5"),
				slog.String("host", originalHost),
				slog.Bool("rewritten", true),
				slog.String("target", dst),
			)
			return ctx, &daSpec
		}
		slog.Warn("unable to parse AddrSpec, skipping...",
			slog.String("host", dst),
			slog.Int("port", request.DestAddr.Port),
		)
	}

	slog.Info("",
		slog.String("client", request.RemoteAddr.String()),
		slog.String("method", "SOCKS5"),
		slog.String("host", originalHost),
		slog.Bool("rewritten", false),
	)
	return ctx, request.DestAddr
}

func runAction(cmd *cobra.Command, args []string) error {
	listenAddress, err := cmd.Flags().GetString("listen-address")
	if err != nil {
		return err
	}

	hostsMap, err := cmd.Flags().GetStringToString("hosts")
	if err != nil {
		return err
	}
	for src, dst := range hostsMap {
		slog.Debug("mapping host",
			slog.String("src", src),
			slog.String("dst", dst),
		)
	}

	switch proxyMode, _ := cmd.Flags().GetString("mode"); proxyMode {
	case "http":
		slog.Debug("starting HTTP proxy",
			slog.String("listen", listenAddress),
		)
		proxy := goproxy.NewProxyHttpServer()
		proxy.Logger = slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo)
		if logLevel.Level() <= slog.LevelDebug {
			proxy.Verbose = true
		}
		proxy.OnRequest().DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				entry := &accessEntry{
					start:  time.Now(),
					client: r.RemoteAddr,
					method: r.Method,
					host:   r.Host,
					path:   r.URL.Path,
				}
				if dst, found := hostsMap[r.Host]; found {
					entry.rewritten = true
					entry.target = dst
					r.URL.Host = dst
				}
				ctx.UserData = entry
				return r, nil
			})
		proxy.OnResponse().DoFunc(
			func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
				if resp == nil {
					return resp
				}
				if entry, ok := ctx.UserData.(*accessEntry); ok {
					attrs := []slog.Attr{
						slog.String("client", entry.client),
						slog.String("method", entry.method),
						slog.String("host", entry.host),
						slog.String("path", entry.path),
						slog.Int("status", resp.StatusCode),
						slog.Int64("size", resp.ContentLength),
						slog.Int64("duration_ms", time.Since(entry.start).Milliseconds()),
						slog.Bool("rewritten", entry.rewritten),
					}
					if entry.rewritten {
						attrs = append(attrs, slog.String("target", entry.target))
					}
					slog.LogAttrs(context.Background(), slog.LevelInfo, "", attrs...)
				}
				return resp
			})
		proxy.OnRequest().HandleConnectFunc(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				h, port, _ := net.SplitHostPort(host)
				if dst, found := hostsMap[h]; found {
					slog.Info("",
						slog.String("client", ctx.Req.RemoteAddr),
						slog.String("method", "CONNECT"),
						slog.String("host", host),
						slog.Bool("rewritten", true),
						slog.String("target", dst),
					)
					return goproxy.OkConnect, net.JoinHostPort(dst, port)
				}
				slog.Info("",
					slog.String("client", ctx.Req.RemoteAddr),
					slog.String("method", "CONNECT"),
					slog.String("host", host),
					slog.Bool("rewritten", false),
				)
				return goproxy.OkConnect, host
			})
		if err := http.ListenAndServe(listenAddress, proxy); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

	case "socks5":
		slog.Debug("starting SOCKS5 proxy",
			slog.String("listen", listenAddress),
		)
		proxy := socks5.NewServer(
			socks5.WithLogger(&socks5Logger{logger: slog.Default()}),
			socks5.WithRewriter(HostRewriter{hostsMap: hostsMap}),
		)
		if err := proxy.ListenAndServe("tcp", listenAddress); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

	default:
		slog.Error("unsupported proxy mode",
			slog.String("mode", proxyMode),
		)
		os.Exit(1)
	}

	return nil
}
