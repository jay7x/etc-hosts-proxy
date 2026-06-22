package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/proxy"
)

var proxyBin string

func TestMain(m *testing.M) {
	out, err := exec.Command("go", "build", "-o", "./etc-hosts-proxy.test", ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
		os.Exit(1)
	}
	proxyBin = "./etc-hosts-proxy.test"
	code := m.Run()
	_ = os.Remove(proxyBin)
	os.Exit(code)
}

func freePort(t testing.TB) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().String()
}

func startProxy(t testing.TB, mode, hosts, listen string) func() {
	t.Helper()
	cmd := exec.Command(proxyBin, "run", "-M", mode, "-L", listen, "-H", hosts)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("proxy start: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	waitForProxy(t, listen, done)

	return func() {
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		if t.Failed() {
			if b := stdout.Bytes(); len(b) > 0 {
				t.Logf("proxy stdout:\n%s", b)
			}
			if b := stderr.Bytes(); len(b) > 0 {
				t.Logf("proxy stderr:\n%s", b)
			}
		}
	}
}

func waitForProxy(t testing.TB, addr string, done <-chan struct{}) {
	t.Helper()
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", addr, 20*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		select {
		case <-done:
			t.Fatal("proxy exited before binding")
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	select {
	case <-done:
		t.Fatal("proxy exited before binding")
	default:
		t.Fatal("proxy did not start within timeout")
	}
}

func startHelloServer(t testing.TB) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	return srv, srv.Listener.Addr().String()
}

func TestHTTP_rewritesHost(t *testing.T) {
	hello, helloAddr := startHelloServer(t)
	defer hello.Close()

	proxyAddr := freePort(t)
	stop := startProxy(t, "http", "hello.example.com="+helloAddr, proxyAddr)
	defer stop()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	defer client.CloseIdleConnections()

	resp, err := client.Get("http://hello.example.com/")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Fatalf("expected 'hello', got %q", body)
	}
}

func TestCONNECT_rewritesHost(t *testing.T) {
	hello, helloAddr := startHelloServer(t)
	defer hello.Close()

	_, port, err := net.SplitHostPort(helloAddr)
	if err != nil {
		t.Fatal(err)
	}
	target := "hello.example.com:" + port
	ip := helloAddr[:strings.LastIndex(helloAddr, ":")]

	proxyAddr := freePort(t)
	stop := startProxy(t, "http", "hello.example.com="+ip, proxyAddr)
	defer stop()

	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	_, _ = fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("CONNECT: expected 200, got %d", resp.StatusCode)
	}

	_, _ = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)

	resp2, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp2.Body.Close() }()
	body, _ := io.ReadAll(resp2.Body)
	if string(body) != "hello" {
		t.Fatalf("expected 'hello', got %q", body)
	}
}

func TestSOCKS5_rewritesHost(t *testing.T) {
	hello, helloAddr := startHelloServer(t)
	defer hello.Close()

	_, port, err := net.SplitHostPort(helloAddr)
	if err != nil {
		t.Fatal(err)
	}
	ip := helloAddr[:strings.LastIndex(helloAddr, ":")]
	target := "test.localhost:" + port

	proxyAddr := freePort(t)
	stop := startProxy(t, "socks5", "test.localhost="+ip, proxyAddr)
	defer stop()

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := dialer.Dial("tcp", target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	_, _ = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", target)

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Fatalf("expected 'hello', got %q", body)
	}
}
