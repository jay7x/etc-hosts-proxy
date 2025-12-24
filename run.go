package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/elazarl/goproxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
)

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

// SOCKS5 destination address rewriter
type HostRewriter struct {
	hostsMap map[string]string
}

func (r HostRewriter) Rewrite(ctx context.Context, request *socks5.Request) (context.Context, *statute.AddrSpec) {
	dst, found := r.hostsMap[request.DestAddr.FQDN]
	if !found {
		dst, found = r.hostsMap[request.DestAddr.IP.String()]
	}

	if found {
		daSpec, err := statute.ParseAddrSpec(net.JoinHostPort(dst, strconv.Itoa(request.DestAddr.Port)))
		if err == nil {
			return ctx, &daSpec
		}
		logrus.Warnf("Unable to parse AddrSpec(%v:%v), skipping...", dst, request.DestAddr.Port)
	}
	return ctx, request.DestAddr
}

// SOCKS5 mapping DNS resolver
type HostResolver struct {
	hostsMap map[string]string
}

func (r HostResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	dst, found := r.hostsMap[name]
	if found {
		return ctx, net.ParseIP(dst), nil
	}

	addr, err := net.ResolveIPAddr("ip", name)
	if err != nil {
		return ctx, nil, err
	}
	return ctx, addr.IP, err
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
		logrus.Debugf("Mapping %s to %s", src, dst)
	}

	switch proxyMode, _ := cmd.Flags().GetString("mode"); proxyMode {
	case "http":
		logrus.Debugf("Starting HTTP proxy on %s", listenAddress)
		proxy := goproxy.NewProxyHttpServer()
		proxy.Logger = logrus.StandardLogger()
		if logrus.GetLevel() >= logrus.DebugLevel {
			proxy.Verbose = true
		}
		proxy.OnRequest().DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				if dst, found := hostsMap[r.Host]; found {
					r.URL.Host = dst
				}
				return r, nil
			})
		proxy.OnRequest().HandleConnectFunc(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				h, port, _ := net.SplitHostPort(host)
				if dst, found := hostsMap[h]; found {
					return goproxy.OkConnect, net.JoinHostPort(dst, port)
				}
				return goproxy.OkConnect, host
			})
		logrus.Fatal(http.ListenAndServe(listenAddress, proxy))

	case "socks5":
		logrus.Debugf("Starting SOCKS5 proxy on %s", listenAddress)
		proxy := socks5.NewServer(
			socks5.WithLogger(logrus.StandardLogger()),
			socks5.WithRewriter(HostRewriter{hostsMap: hostsMap}),
			socks5.WithResolver(HostResolver{hostsMap: hostsMap}),
		)
		if err := proxy.ListenAndServe("tcp", listenAddress); err != nil {
			logrus.Fatal(err)
		}

	default:
		logrus.Fatalf("Unsupported proxy mode %v", proxyMode)
	}

	return nil
}
