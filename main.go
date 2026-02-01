package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/xconnect/xconnect-go/internal/daemon"
	"github.com/xconnect/xconnect-go/internal/discovery"
	"github.com/xconnect/xconnect-go/internal/server"
	clipsync "github.com/xconnect/xconnect-go/internal/sync"
)

var (
	addr         = flag.String("addr", ":8315", "address to listen on")
	useTsnet     = flag.Bool("tsnet", false, "use embedded Tailscale (tsnet); if false, assume system Tailscale")
	hostname     = flag.String("hostname", "xconnect", "hostname on tailnet (used when -tsnet)")
	authKey      = flag.String("authkey", "", "Tailscale auth key (used when -tsnet); or set TS_AUTHKEY")
	enableSync   = flag.Bool("sync", false, "enable clipboard auto-sync: broadcast local copy to other devices")
	syncInterval = flag.Duration("sync-interval", time.Second, "clipboard poll interval when -sync")
	apiToken     = flag.String("api-token", "", "Tailscale API token for peer discovery (or TAILSCALE_API_TOKEN)")
	peersList    = flag.String("peers", "", "comma-separated peer hostnames or IPs (overrides discovery when -sync)")
	daemonMode   = flag.Bool("daemon", false, "run in background (service mode); logs to file")
	logFile      = flag.String("log-file", "", "log file path (default: platform-specific, e.g. %%LocalAppData%%\\XConnect\\logs on Windows)")
)

func main() {
	flag.Parse()

	// Daemon: parent starts child and exits; child runs with stderr = log file
	if *daemonMode && !daemon.IsDaemonChild() {
		if err := daemon.RunInBackground(*logFile); err != nil {
			log.Fatalf("daemon: %v", err)
		}
		return
	}
	if daemon.IsDaemonChild() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.Ldate | log.Ltime)
	} else if *logFile != "" {
		f, err := daemon.SetupLog(*logFile, true)
		if err != nil {
			log.Fatalf("log-file: %v", err)
		}
		defer f.Close()
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type lastReceivedState struct {
	mu   sync.Mutex
	val  string
}

func (s *lastReceivedState) Get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.val
}

func (s *lastReceivedState) Set(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.val = v
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

	lastReceived := &lastReceivedState{}
	handlerOpts := &server.HandlerOpts{
		OnClipboardReceivedFromNetwork: lastReceived.Set,
	}
	handler := server.NewHandler(handlerOpts)

	if *enableSync {
		ctx := context.Background()
		port := *addr
		if strings.HasPrefix(port, ":") {
			port = port[1:]
		}
		selfHost := *hostname
		if !*useTsnet {
			ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
			if t, _ := os.LookupEnv("TAILSCALE_API_TOKEN"); *apiToken == "" {
				*apiToken = t
			}
			if s, _, err := discovery.SelfAndPeers(ctx2, *apiToken); err == nil && s != "" {
				selfHost = s
			}
			cancel()
		}
		getPeers := func() []string {
			var urls []string
			if *peersList != "" {
				for _, p := range strings.Split(*peersList, ",") {
					p = strings.TrimSpace(p)
					if p != "" && p != selfHost {
						urls = append(urls, "http://"+p+":"+port)
					}
				}
				return urls
			}
			ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, peers, err := discovery.SelfAndPeers(ctx2, *apiToken)
			cancel()
			if err != nil {
				return nil
			}
			for _, d := range peers {
				if d.HostName == selfHost {
					continue
				}
				u := discovery.BaseURL(d, port)
				if u != "" {
					urls = append(urls, u)
				}
			}
			return urls
		}
		getClipboard := func() string {
			s, _ := clipboard.ReadAll()
			return s
		}
		getFromHost := func() string { return selfHost }
		go clipsync.ClipboardSync(ctx, clipsync.Options{
			Interval:        *syncInterval,
			GetClipboard:   getClipboard,
			GetLastReceived: lastReceived.Get,
			GetPeers:        getPeers,
			GetFromHost:     getFromHost,
			HTTPClient:      &http.Client{Timeout: 10 * time.Second},
		})
		log.Printf("clipboard auto-sync enabled (broadcast to peers on copy)")
	}

	log.Printf("xconnect listening on %s (tsnet=%v)", *addr, *useTsnet)
	return http.Serve(ln, handler)
}
