//go:build !windows

package nnet

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestSmoke starts an echo server and runs several hundred ping-pong
// round trips over a single client connection, then shuts the server down.
func TestSmoke(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	addr := ln.Addr().String()

	onRequest := func(ctx context.Context, conn Connection) error {
		reader := conn.Reader()
		msg, err := reader.Next(reader.Len())
		if err != nil {
			return err
		}
		if _, err = conn.Write(msg); err != nil {
			return err
		}
		return nil
	}

	el, err := NewEventLoop(onRequest)
	if err != nil {
		t.Fatalf("NewEventLoop failed: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- el.Serve(ln)
	}()

	conn, err := DialConnection("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("DialConnection failed: %v", err)
	}
	defer conn.Close()

	const rounds = 500
	for i := 0; i < rounds; i++ {
		if _, err = conn.Write([]byte("ping")); err != nil {
			t.Fatalf("round %d write failed: %v", i, err)
		}
		buf := make([]byte, 4)
		if _, err = conn.Read(buf); err != nil { // net.Conn.Read blocks until data arrives
			t.Fatalf("round %d read failed: %v", i, err)
		}
		if string(buf) != "ping" {
			t.Fatalf("round %d got %q, want %q", i, buf, "ping")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = el.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	select {
	case err = <-serveDone:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after Shutdown")
	}
}
