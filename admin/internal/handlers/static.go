package handlers

import (
	"net/http"
	"path/filepath"
)

// StaticHandler serves static files
type StaticHandler struct {
	staticDir string
}

// NewStaticHandler creates a new static handler
func NewStaticHandler(staticDir string) *StaticHandler {
	return &StaticHandler{
		staticDir: staticDir,
	}
}

// ServeHTTP serves static files
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for root path
	if r.URL.Path == "/" {
		http.ServeFile(w, r, filepath.Join(h.staticDir, "index.html"))
		return
	}

	// Serve login.html for /login path
	if r.URL.Path == "/login" || r.URL.Path == "/login.html" {
		http.ServeFile(w, r, filepath.Join(h.staticDir, "login.html"))
		return
	}

	// Serve static files
	http.FileServer(http.Dir(h.staticDir)).ServeHTTP(w, r)
}