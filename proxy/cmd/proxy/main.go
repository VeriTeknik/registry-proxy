package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/veriteknik/registry-proxy/internal/cache"
	"github.com/veriteknik/registry-proxy/internal/handlers"
)

func main() {
	// Configuration
	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8090"
	}

	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		registryURL = "http://registry:8080"
	}

	cacheExpiration := 5 * time.Minute
	cacheCleanup := 10 * time.Minute

	// Initialize cache
	proxyCache := cache.NewCache(cacheExpiration, cacheCleanup)

	// Initialize handlers
	serversHandler := handlers.NewServersHandler(registryURL, proxyCache)
	passthroughHandler, err := handlers.NewPassthroughHandler(registryURL)
	if err != nil {
		log.Fatalf("Failed to create passthrough handler: %v", err)
	}

	// Setup routes
	mux := http.NewServeMux()
	
	// Health check (internal)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"registry-proxy"}`)
	})

	// Registry-compatible health check
	mux.HandleFunc("/v0/health", func(w http.ResponseWriter, r *http.Request) {
		// Proxy to upstream to get GitHub client ID
		passthroughHandler.ProxySpecificEndpoint().ServeHTTP(w, r)
	})

	// Enriched servers endpoint (only for GET requests)
	mux.HandleFunc("/v0/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v0/servers" {
			// Use our enriched handler for list
			serversHandler.HandleList(w, r)
		} else {
			// Proxy everything else (like /v0/servers/{id})
			passthroughHandler.ProxySpecificEndpoint().ServeHTTP(w, r)
		}
	})

	// Publish endpoint (proxy to upstream)
	mux.HandleFunc("/v0/publish", passthroughHandler.ProxySpecificEndpoint())
	
	// Cache refresh endpoint (our custom endpoint)
	mux.HandleFunc("/v0/cache/refresh", serversHandler.HandleRefresh)

	// Catch-all for any other endpoints
	mux.HandleFunc("/", passthroughHandler.ProxySpecificEndpoint())

	// CORS middleware
	handler := corsMiddleware(mux)

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting registry proxy on %s", addr)
	log.Printf("Upstream registry: %s", registryURL)
	log.Printf("Cache expiration: %v", cacheExpiration)
	
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}