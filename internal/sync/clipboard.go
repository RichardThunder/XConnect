package sync

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

// ClipboardSync runs a loop that polls local clipboard and broadcasts to peers when it changes.
// getClipboard returns current local clipboard content; getLastReceived returns the last content
// we received from the network (so we don't re-broadcast it).
func ClipboardSync(ctx context.Context, opts Options) {
	interval := opts.Interval
	if interval <= 0 {
		interval = time.Second
	}
	var lastBroadcasted string
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		current := opts.GetClipboard()
		lastReceived := opts.GetLastReceived()
		if current == "" {
			continue
		}
		if current == lastReceived {
			continue
		}
		if current == lastBroadcasted {
			continue
		}
		peers := opts.GetPeers()
		if len(peers) == 0 {
			continue
		}
		ok := true
		for _, baseURL := range peers {
			url := strings.TrimSuffix(baseURL, "/") + "/clipboard"
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(current)))
			if err != nil {
				log.Printf("sync: new request: %v", err)
				ok = false
				continue
			}
			req.Header.Set("Content-Type", "text/plain; charset=utf-8")
			resp, err := opts.HTTPClient.Do(req)
			if err != nil {
				log.Printf("sync: POST %s: %v", url, err)
				ok = false
				continue
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
				log.Printf("sync: POST %s: %s", url, resp.Status)
				ok = false
			}
		}
		if ok {
			lastBroadcasted = current
		}
	}
}

// Options configures ClipboardSync.
type Options struct {
	Interval       time.Duration
	GetClipboard   func() string
	GetLastReceived func() string
	GetPeers       func() []string
	HTTPClient     *http.Client
}
