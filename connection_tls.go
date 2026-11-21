//go:build !windows

package nnet

import (
	"context"
	"crypto/tls"
	"sync/atomic"
)

const (
	tlsIdle     = 0
	tlsCallback = 1
	tlsClosing  = 2
)

type tlsConnection struct {
	Connection
	raw    *connection
	conn   *tls.Conn
	reader *zcReader
	writer *zcWriter
	opts   *options

	onRequest  atomic.Value
	phase      atomic.Uint32
	handshaken bool
}

var _ Connection = (*tlsConnection)(nil)

func newTLSConnection(raw *connection, opts *options) *tlsConnection {
	conn := tls.Server(raw, opts.tlsConfig)
	secure := &tlsConnection{
		Connection: raw,
		raw:        raw,
		conn:       conn,
		reader:     newZCReader(conn),
		writer:     newZCWriter(conn),
		opts:       opts,
	}
	if opts.onRequest != nil {
		secure.onRequest.Store(opts.onRequest)
	}
	return secure
}

func (c *tlsConnection) connect(ctx context.Context, _ Connection) context.Context {
	err := c.conn.HandshakeContext(ctx)
	if err != nil {
		_ = c.raw.Close()
		return ctx
	}
	c.handshaken = true
	if c.opts.onConnect != nil && c.phase.CompareAndSwap(tlsIdle, tlsCallback) {
		defer c.phase.Store(tlsIdle)
		if c.IsActive() {
			return c.opts.onConnect(ctx, c)
		}
	}
	return ctx
}

func (c *tlsConnection) disconnect(ctx context.Context) {
	if c.handshaken && c.opts.onDisconnect != nil {
		c.opts.onDisconnect(ctx, c)
	}
}

func (c *tlsConnection) process(ctx context.Context) {
	defer c.phase.CompareAndSwap(tlsCallback, tlsIdle)
	for c.phase.Load() != tlsClosing && c.raw.status(closing) != user && c.onRequest.Load() != nil {
		// TLS may already hold plaintext after the TCP buffer is drained
		err := c.reader.waitRead(1)
		if err != nil {
			_ = c.raw.Close()
			return
		}
		if !c.phase.CompareAndSwap(tlsIdle, tlsCallback) || c.raw.status(closing) == user {
			return
		}
		onRequest := c.onRequest.Load().(OnRequest)
		err = onRequest(ctx, c)
		c.phase.Store(tlsIdle)
		if err != nil {
			_ = c.raw.Close()
			return
		}
	}
}

func (c *tlsConnection) Reader() Reader { return c.reader }

func (c *tlsConnection) Writer() Writer { return c.writer }

func (c *tlsConnection) Read(p []byte) (int, error) {
	if c.reader.Len() > 0 {
		return c.reader.buf.readCopy(p), nil
	}
	return c.conn.Read(p)
}

func (c *tlsConnection) Write(p []byte) (int, error) { return c.conn.Write(p) }

func (c *tlsConnection) Close() error { return c.conn.Close() }

func (c *tlsConnection) SetOnRequest(handler OnRequest) error {
	if handler != nil {
		c.onRequest.Store(handler)
		if c.raw.getState() == connStateConnected {
			c.raw.onProcess(nil, nil)
		}
	}
	return nil
}

func (c *tlsConnection) AddCloseCallback(callback CloseCallback) error {
	if callback == nil {
		return nil
	}
	return c.raw.AddCloseCallback(func(_ Connection) error {
		return callback(c)
	})
}
