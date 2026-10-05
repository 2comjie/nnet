// Copyright 2022 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nnet

import (
	"net"
	"time"
)

type CloseCallback func(connection Connection) error

type Connection interface {
	net.Conn
	Reader() Reader
	Writer() Writer
	IsActive() bool

	// SetReadTimeout sets the timeout for future Read calls wait.
	// A zero value for timeout means Reader will not timeout.
	SetReadTimeout(timeout time.Duration) error

	// SetWriteTimeout sets the timeout for future Write calls wait.
	// A zero value for timeout means Writer will not timeout.
	SetWriteTimeout(timeout time.Duration) error

	// SetIdleTimeout sets the idle timeout of connections by enabling TCP KeepAlive
	// and setting the KeepAlive interval to the given timeout duration.
	// NOTE: Despite its name, this does not track application-level idle time.
	// It configures OS-level TCP KeepAlive to detect dead peers on idle connections.
	// The name is kept for backward compatibility.
	SetIdleTimeout(timeout time.Duration) error

	// SetOnRequest can set or replace the OnRequest method for a connection, but can't be set to nil.
	// Although SetOnRequest avoids data race, it should still be used before transmitting data.
	// Replacing OnRequest while processing data may cause unexpected behavior and results.
	// Generally, the server side should uniformly set the OnRequest method for each connection via NewEventLoop,
	// which is set when the connection is initialized.
	// On the client side, if necessary, make sure that OnRequest is set before sending data.
	SetOnRequest(on OnRequest) error

	// AddCloseCallback can add hangup callback for a connection, which will be called when connection closing.
	// This is very useful for cleaning up idle connections. For instance, you can use callbacks to clean up
	// the local resources, which bound to the idle connection, when hangup by the peer. No need another goroutine
	// to polling check connection status.
	AddCloseCallback(closeCallback CloseCallback) error
}

type Conn interface {
	net.Conn
	Fd() (fd int)
}

type Listener interface {
	net.Listener

	Fd() (fd int)
}

// Dialer extends net.Dialer's API, just for interface compatibility.
// DialConnection is recommended, but of course all functions are practically the same.
// The returned net.Conn can be directly asserted as Connection if error is nil.
type Dialer interface {
	DialConnection(network, address string, timeout time.Duration) (connection Connection, err error)

	DialTimeout(network, address string, timeout time.Duration) (conn net.Conn, err error)
}
