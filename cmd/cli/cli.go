package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/xconnect/xconnect-go/internal/clipboard"
	"github.com/xconnect/xconnect-go/internal/discovery"
)

var (
	port     = flag.String("port", "8315", "peer service port")
	apiToken = flag.String("api-token", "", "Tailscale API token for device list (or TAILSCALE_API_TOKEN)")
)

func main() {
	flag.Parse()
	if apiToken == nil || *apiToken == "" {
		*apiToken = os.Getenv("TAILSCALE_API_TOKEN")
	}
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "list":
		runList(rest)
	case "push":
		runPush(rest)
	case "pull":
		runPull(rest)
	case "message":
		runMessage(rest)
	case "file":
		runFile(rest)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  xconnect list                    list tailnet devices
  xconnect push <peer>             push local clipboard to peer
  xconnect pull <peer>             pull peer clipboard to local
  xconnect message <peer> <text>   send message (text) to peer
  xconnect file <peer> <path>      send file to peer

Peers: hostname (MagicDNS) or 100.x.x.x. Port defaults to %s.
`, *port)
}

func baseURL(peer string) string {
	if *port == "" {
		*port = "8315"
	}
	return "http://" + peer + ":" + *port
}

func runList(rest []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	devices, err := discovery.Devices(ctx, *apiToken)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	for _, d := range devices {
		url := discovery.BaseURL(d, *port)
		if url == "" {
			continue
		}
		fmt.Printf("%s\t%s\n", d.HostName, url)
	}
}

func runPush(rest []string) {
	if len(rest) < 1 {
		log.Fatal("usage: xconnect push <peer>")
	}
	peer := rest[0]
	text, err := clipboard.ReadAll()
	if err != nil {
		log.Fatalf("read clipboard: %v", err)
	}
	url := baseURL(peer) + "/clipboard"
	resp, err := http.Post(url, "text/plain; charset=utf-8", bytes.NewReader([]byte(text)))
	if err != nil {
		log.Fatalf("push: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("push: %s %s", resp.Status, string(body))
	}
	fmt.Println("clipboard pushed to", peer)
}

func runPull(rest []string) {
	if len(rest) < 1 {
		log.Fatal("usage: xconnect pull <peer>")
	}
	peer := rest[0]
	url := baseURL(peer) + "/clipboard"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("pull: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("pull: %s %s", resp.Status, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("pull: %v", err)
	}
	if err := clipboard.WriteAll(string(body)); err != nil {
		log.Fatalf("write clipboard: %v", err)
	}
	fmt.Println("clipboard pulled from", peer)
}

func runMessage(rest []string) {
	if len(rest) < 2 {
		log.Fatal("usage: xconnect message <peer> <text>")
	}
	peer := rest[0]
	text := rest[1]
	url := baseURL(peer) + "/message"
	payload, _ := json.Marshal(map[string]string{"text": text})
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Fatalf("message: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("message: %s %s", resp.Status, string(body))
	}
	fmt.Println("message sent to", peer)
}

func runFile(rest []string) {
	if len(rest) < 2 {
		log.Fatal("usage: xconnect file <peer> <path>")
	}
	peer := rest[0]
	path := rest[1]
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open file: %v", err)
	}
	defer f.Close()
	url := baseURL(peer) + "/files"
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		log.Fatalf("form: %v", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		log.Fatalf("copy: %v", err)
	}
	mw.Close()
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		log.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("file: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("file: %s %s", resp.Status, string(body))
	}
	var out struct {
		ID string `json:"file_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Fatalf("decode: %v", err)
	}
	fmt.Printf("file uploaded to %s, id=%s\n", peer, out.ID)
}
