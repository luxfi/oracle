// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zaptransport

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	_, p, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(p)
	return port
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestZAPTransport_BroadcastObservation exercises the end-to-end ZAP path:
// node A broadcasts a JSON-encoded Observation, node B's handler receives it.
func TestZAPTransport_BroadcastObservation(t *testing.T) {
	portA := freePort(t)
	portB := freePort(t)

	a, err := New(Config{NodeID: "oracle-a", Port: portA, NoDiscovery: true, Logger: quietLogger()})
	if err != nil {
		t.Fatalf("new a: %v", err)
	}
	b, err := New(Config{NodeID: "oracle-b", Port: portB, NoDiscovery: true, Logger: quietLogger()})
	if err != nil {
		t.Fatalf("new b: %v", err)
	}

	if err := a.Start(); err != nil {
		t.Fatalf("start a: %v", err)
	}
	defer a.Stop()
	if err := b.Start(); err != nil {
		t.Fatalf("start b: %v", err)
	}
	defer b.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	var received []byte
	var mu sync.Mutex
	b.HandleObservation(func(_ context.Context, _ string, payload []byte) error {
		mu.Lock()
		received = append([]byte(nil), payload...)
		mu.Unlock()
		wg.Done()
		return nil
	})

	if err := a.ConnectDirect("127.0.0.1:" + strconv.Itoa(portB)); err != nil {
		t.Fatalf("connect a->b: %v", err)
	}

	obs := map[string]any{
		"feedId":    "lux/usd",
		"value":     "100.5",
		"scheme":    1,
		"signature": "deadbeef",
	}
	body, _ := json.Marshal(obs)

	errs := a.BroadcastObservation(context.Background(), body)
	for peer, err := range errs {
		if err != nil {
			t.Fatalf("broadcast to %s: %v", peer, err)
		}
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for observation")
	}

	mu.Lock()
	defer mu.Unlock()
	if string(received) != string(body) {
		t.Fatalf("observation payload mismatch: got %q want %q", received, body)
	}
}

// TestZAPTransport_BroadcastRecord exercises the executor record broadcast.
func TestZAPTransport_BroadcastRecord(t *testing.T) {
	portA := freePort(t)
	portB := freePort(t)

	a, _ := New(Config{NodeID: "oracle-a", Port: portA, NoDiscovery: true, Logger: quietLogger()})
	b, _ := New(Config{NodeID: "oracle-b", Port: portB, NoDiscovery: true, Logger: quietLogger()})
	if err := a.Start(); err != nil {
		t.Fatalf("start a: %v", err)
	}
	defer a.Stop()
	if err := b.Start(); err != nil {
		t.Fatalf("start b: %v", err)
	}
	defer b.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	var received []byte
	var mu sync.Mutex
	b.HandleRecord(func(_ context.Context, _ string, payload []byte) error {
		mu.Lock()
		received = append([]byte(nil), payload...)
		mu.Unlock()
		wg.Done()
		return nil
	})

	if err := a.ConnectDirect("127.0.0.1:" + strconv.Itoa(portB)); err != nil {
		t.Fatalf("connect: %v", err)
	}

	rec := map[string]any{"endpoint": "https://x.test", "scheme": 1}
	body, _ := json.Marshal(rec)
	for peer, err := range a.BroadcastRecord(context.Background(), body) {
		if err != nil {
			t.Fatalf("broadcast to %s: %v", peer, err)
		}
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for record")
	}
	mu.Lock()
	if string(received) != string(body) {
		t.Fatalf("record mismatch")
	}
	mu.Unlock()
}

func TestZAPTransport_NoNodeIDRejected(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatalf("expected empty NodeID to be refused")
	}
}
