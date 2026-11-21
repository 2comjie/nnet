//go:build !windows

package nnet

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestShutdownWithServe(t *testing.T) {
	cases := []struct {
		name          string
		shutdownFirst bool
	}{
		{name: "before Serve", shutdownFirst: true},
		{name: "concurrent with Serve"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loop, err := NewEventLoop(nil)
			if err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			addr := listener.Addr().String()

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			start := make(chan struct{})
			shutdownDone := make(chan struct{})
			serveDone := make(chan error, 1)
			var shutdownErr error
			go func() {
				<-start
				shutdownErr = loop.Shutdown(ctx)
				close(shutdownDone)
			}()
			go func() {
				<-start
				if tc.shutdownFirst {
					<-shutdownDone
				}
				serveDone <- loop.Serve(listener)
			}()
			close(start)

			select {
			case <-shutdownDone:
				if shutdownErr != nil {
					t.Fatalf("Shutdown returned error: %v", shutdownErr)
				}
			case <-ctx.Done():
				t.Fatal("Shutdown did not return")
			}
			select {
			case err := <-serveDone:
				if err != nil {
					t.Fatalf("Serve returned error: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("Serve did not return after Shutdown")
			}

			reopened, err := net.Listen("tcp", addr)
			if err != nil {
				t.Fatalf("cannot rebind %s after Shutdown: %v", addr, err)
			}
			defer reopened.Close()
		})
	}
}
