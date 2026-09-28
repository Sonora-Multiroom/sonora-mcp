package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tiger-seo/sonora-mcp/internal/tools"
)

// freeAddr returns a loopback address with a port nothing listens on.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// waitListening waits until addr accepts connections.
func waitListening(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server not listening on %s: %v", addr, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRunDrainsInFlightCallsOnShutdown(t *testing.T) {
	hubStarted := make(chan struct{}, 1)
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hubStarted <- struct{}{}
		time.Sleep(300 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"muted":true}`))
	}))
	t.Cleanup(hub.Close)

	s := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: "test"}, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	tools.Register(s, client, hub.URL)
	addr := freeAddr(t)
	srv := &http.Server{Addr: addr, Handler: Handler(s, client, hub.URL)}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- Run(ctx, srv) }()
	waitListening(t, addr)

	mc := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := mc.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: "http://" + addr + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()

	type result struct {
		res *mcp.CallToolResult
		err error
	}
	call := make(chan result, 1)
	go func() {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "getMasterMute", Arguments: map[string]any{}})
		call <- result{res, err}
	}()
	<-hubStarted
	start := time.Now()
	cancel() // simulated stop signal

	r := <-call
	if r.err != nil {
		t.Fatalf("in-flight call: %v", r.err)
	}
	if r.res.IsError {
		t.Fatalf("in-flight call failed: %v", r.res.Content)
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run = %v, want nil", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Run did not return within 6s")
	}
	if d := time.Since(start); d > 6*time.Second {
		t.Errorf("shutdown took %v", d)
	}
	if c, err := net.Dial("tcp", addr); err == nil {
		c.Close()
		t.Error("new connection accepted after shutdown")
	}
}

func TestRunListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	srv := &http.Server{Addr: ln.Addr().String(), Handler: http.NotFoundHandler()}
	if err := Run(context.Background(), srv); err == nil {
		t.Fatal("Run on a taken port = nil, want error")
	}
}
