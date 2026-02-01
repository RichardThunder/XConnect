package server

import (
	"net"
)

// ListenSystem binds on the given address using the system network stack.
// When Tailscale is installed, traffic to this process over the Tailscale
// interface (MagicDNS hostname or 100.x.x.x) will reach this listener.
func ListenSystem(addr string) (Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &wrapListener{Listener: ln}, nil
}

type wrapListener struct {
	net.Listener
}

func (w *wrapListener) Close() error {
	return w.Listener.Close()
}
