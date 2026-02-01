package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/xconnect/xconnect-go/internal/server"
)

var (
	addr     = flag.String("addr", ":8315", "address to listen on")
	useTsnet = flag.Bool("tsnet", false, "use embedded Tailscale (tsnet); if false, assume system Tailscale")
	hostname = flag.String("hostname", "xconnect", "hostname on tailnet (used when -tsnet)")
	authKey  = flag.String("authkey", "", "Tailscale auth key (used when -tsnet); or set TS_AUTHKEY")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var ln server.Listener
	var err error
	if *useTsnet {
		ln, err = server.ListenTsnet(*hostname, *authKey, *addr)
		if err != nil {
			return err
		}
		defer ln.Close()
	} else {
		ln, err = server.ListenSystem(*addr)
		if err != nil {
			return err
		}
		defer ln.Close()
	}

	handler := server.NewHandler()
	log.Printf("xconnect listening on %s (tsnet=%v)", *addr, *useTsnet)
	return http.Serve(ln, handler)
}
