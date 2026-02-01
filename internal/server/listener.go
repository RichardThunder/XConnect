package server

import "net"

// Listener is a net.Listener that can be closed.
type Listener interface {
	net.Listener
	Close() error
}
