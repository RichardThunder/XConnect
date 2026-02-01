package server

import (
	"net"
	"os"

	"tailscale.com/tsnet"
)

// ListenTsnet starts an embedded Tailscale server and listens on the given address
// only on the Tailscale network. hostname is the MagicDNS name; authKey can be empty
// if TS_AUTHKEY env is set.
func ListenTsnet(hostname, authKey, addr string) (Listener, error) {
	srv := &tsnet.Server{
		Hostname: hostname,
		AuthKey:  authKey,
	}
	if srv.AuthKey == "" {
		srv.AuthKey = os.Getenv("TS_AUTHKEY")
	}
	if err := srv.Start(); err != nil {
		return nil, err
	}
	ln, err := srv.Listen("tcp", addr)
	if err != nil {
		srv.Close()
		return nil, err
	}
	return &tsnetListener{Server: srv, Listener: ln}, nil
}

type tsnetListener struct {
	*tsnet.Server
	net.Listener
}

func (t *tsnetListener) Close() error {
	t.Listener.Close()
	return t.Server.Close()
}
