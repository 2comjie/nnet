package nnet

import (
	"context"
	"net"
)

type EventLoop interface {
	// Serve closes ln immediately if Shutdown was already called
	Serve(ln net.Listener) error
	// Shutdown permanently stops this loop, including before Serve starts
	Shutdown(ctx context.Context) error
}

type OnPrepare func(connection Connection) context.Context

type OnConnect func(ctx context.Context, connection Connection) context.Context

type OnDisconnect func(ctx context.Context, connection Connection)

type OnRequest func(ctx context.Context, connection Connection) error
