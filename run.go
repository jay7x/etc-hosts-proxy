package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	runCommand.Flags().StringToStringP("hosts", "H", map[string]string{}, "<host>=<ip> pairs to resolve <host> to <ip>")
	runCommand.Flags().StringP("mode", "M", "http", "Mode to start proxy in (http or socks5)")
	runCommand.Flags().StringP("listen-address", "L", "127.0.0.1:8080", "[<host>]:<port> to listen for proxy requests on")
	return runCommand
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
	for host, ip := range hostsMap {
		logrus.Debugf("Mapping %s to %s", host, ip)
	}

	switch proxyMode, _ := cmd.Flags().GetString("mode"); proxyMode {
	case "http":
		logrus.Debugln("Starting HTTP proxy...")
		proxy := goproxy.NewProxyHttpServer()
		proxy.Logger = logrus.StandardLogger()
		if logrus.GetLevel() >= logrus.DebugLevel {
			proxy.Verbose = true
		}
		proxy.OnRequest().DoFunc(
			func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				if ip, found := hostsMap[r.Host]; found {
					r.URL.Host = ip
				}
				return r, nil
			})
		proxy.OnRequest().HandleConnectFunc(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				h, port, _ := net.SplitHostPort(host)
				if ip, found := hostsMap[h]; found {
					return goproxy.OkConnect, net.JoinHostPort(ip, port)
				}
				return goproxy.OkConnect, host
			})
		logrus.Fatal(http.ListenAndServe(listenAddress, proxy))
	case "socks5":
		logrus.Debugln("Starting SOCKS5 proxy...")
		logrus.Fatalln("SOCKS5 proxy is not implemented yet...")
	default:
		logrus.Fatalf("Unsupported proxy mode %v", proxyMode)
	}

	return nil
}
