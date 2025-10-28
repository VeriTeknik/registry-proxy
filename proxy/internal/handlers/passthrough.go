package handlers

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/veriteknik/registry-proxy/internal/cache"
)

// PassthroughHandler proxies requests to the upstream registry
type PassthroughHandler struct {
	upstreamURL *url.URL
	proxy       *httputil.ReverseProxy
	cache       *cache.Cache
}

// NewPassthroughHandler creates a new passthrough handler
func NewPassthroughHandler(upstreamURL string, cache *cache.Cache) (*PassthroughHandler, error) {
	u, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	
	// Customize the director to handle path correctly
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = u.Host
		req.URL.Host = u.Host
		req.URL.Scheme = u.Scheme
	}

	return &PassthroughHandler{
		upstreamURL: u,
		proxy:       proxy,
		cache:       cache,
	}, nil
}

// ServeHTTP handles the proxying
func (h *PassthroughHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For specific endpoints we handle ourselves, don't proxy
	if r.URL.Path == "/v0/servers" && r.Method == http.MethodGet {
		// This is handled by our enriched handler
		return
	}

	// Proxy all other requests
	h.proxy.ServeHTTP(w, r)
}

// ProxySpecificEndpoint creates a handler for specific endpoint proxying
func (h *PassthroughHandler) ProxySpecificEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Build upstream URL
		upstreamPath := r.URL.Path
		if r.URL.RawQuery != "" {
			upstreamPath += "?" + r.URL.RawQuery
		}

		upstreamURL := h.upstreamURL.String() + upstreamPath

		// Create new request
		proxyReq, err := http.NewRequest(r.Method, upstreamURL, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Copy headers
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Make request
		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "Failed to proxy request", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Check if this is a successful publish request
		if r.Method == http.MethodPost && r.URL.Path == "/v0/publish" && resp.StatusCode == http.StatusCreated {
			// Clear cache on successful publish
			if h.cache != nil {
				h.cache.Clear()
				log.Printf("Cache cleared after successful publish to %s", r.URL.Path)
			}
		}

		// Copy response headers
		for key, values := range resp.Header {
			// Skip hop-by-hop headers
			if isHopByHopHeader(key) {
				continue
			}
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Set status code
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		if _, err := io.Copy(w, resp.Body); err != nil {
			// Log error but can't return it since headers are already sent
			log.Printf("Error copying response body: %v", err)
		}
	}
}

// isHopByHopHeader checks if a header is hop-by-hop
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	header = strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}