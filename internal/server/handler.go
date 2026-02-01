package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xconnect/xconnect-go/internal/clipboard"
)

const defaultFileDir = "xconnect-files"

// HandlerOpts optionally configures the handler (e.g. for clipboard sync).
type HandlerOpts struct {
	// OnClipboardReceivedFromNetwork is called when we write clipboard content received from a peer.
	// Used by sync to avoid re-broadcasting that content.
	OnClipboardReceivedFromNetwork func(content string)
}

// NewHandler returns an http.Handler for the xconnect API.
// If opts is nil, no optional behaviour is used.
func NewHandler(opts *HandlerOpts) http.Handler {
	mux := http.NewServeMux()
	h := &handler{
		fileDir: defaultFileDir,
		files:   make(map[string]string),
		opts:    opts,
	}
	mux.HandleFunc("GET /clipboard", h.getClipboard)
	mux.HandleFunc("POST /clipboard", h.postClipboard)
	mux.HandleFunc("POST /files", h.postFiles)
	mux.HandleFunc("GET /files/", h.getFile)
	mux.HandleFunc("POST /message", h.postMessage)
	mux.HandleFunc("GET /ws", h.serveWebSocket)
	return mux
}

type handler struct {
	fileDir string
	mu      sync.Mutex
	files   map[string]string
	opts    *HandlerOpts
}

func (h *handler) getClipboard(w http.ResponseWriter, r *http.Request) {
	text, err := clipboard.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(text))
}

func (h *handler) postClipboard(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		// Optional: image or file in form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Prefer text field
		if t := r.FormValue("text"); t != "" {
			if err := clipboard.WriteAll(t); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if h.opts != nil && h.opts.OnClipboardReceivedFromNetwork != nil {
				h.opts.OnClipboardReceivedFromNetwork(t)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// TODO: image clipboard if needed per platform
	}
	// Plain text body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	content := string(body)
	if err := clipboard.WriteAll(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if h.opts != nil && h.opts.OnClipboardReceivedFromNetwork != nil {
		h.opts.OnClipboardReceivedFromNetwork(content)
	}
	w.WriteHeader(http.StatusNoContent)
}

type fileResponse struct {
	ID string `json:"file_id"`
}

func (h *handler) postFiles(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100MB
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(h.fileDir, 0700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var id string
	var path string
	var filename string
	for name, headers := range r.MultipartForm.File {
		_ = name
		for _, header := range headers {
			filename = header.Filename
			f, err := header.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b := make([]byte, 8)
			if _, err := rand.Read(b); err != nil {
				f.Close()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			id = hex.EncodeToString(b)
			ext := filepath.Ext(header.Filename)
			if ext == "" {
				ext = ".bin"
			}
			path = filepath.Join(h.fileDir, id+ext)
			out, err := os.Create(path)
			if err != nil {
				f.Close()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, err = io.Copy(out, f)
			out.Close()
			f.Close()
			if err != nil {
				os.Remove(path)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			break
		}
		if id != "" {
			break
		}
	}
	if id == "" {
		http.Error(w, "no file in request", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.files[id] = path
	if filename != "" {
		h.files[id+":name"] = filename
	}
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileResponse{ID: id})
}

func (h *handler) getFile(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/files/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		http.Error(w, "missing file id", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	path, ok := h.files[id]
	origName := h.files[id+":name"]
	h.mu.Unlock()
	if !ok {
		path = filepath.Join(h.fileDir, id)
		if _, err := os.Stat(path); err != nil {
			entries, _ := os.ReadDir(h.fileDir)
			for _, e := range entries {
				base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				if base == id {
					path = filepath.Join(h.fileDir, e.Name())
					origName = e.Name()
					break
				}
			}
		}
	}
	if path == "" {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()
	info, _ := f.Stat()
	name := origName
	if name == "" {
		name = info.Name()
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	http.ServeContent(w, r, name, info.ModTime(), f)
}

type messageRequest struct {
	Text string `json:"text"`
}

func (h *handler) postMessage(w http.ResponseWriter, r *http.Request) {
	var req messageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Write to clipboard as the "message" delivery
	if req.Text != "" {
		_ = clipboard.WriteAll(req.Text)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	// Simple WebSocket upgrade placeholder; full impl would use gorilla/websocket or nhooyr.io
	http.Error(w, "WebSocket not implemented; use POST /clipboard or POST /message", http.StatusNotImplemented)
}
