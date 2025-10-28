package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/veriteknik/registry-proxy/internal/cache"
	"github.com/veriteknik/registry-proxy/internal/db"
	"github.com/veriteknik/registry-proxy/internal/handlers"
	"github.com/veriteknik/registry-proxy/internal/middleware"
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

	// Initialize database connections
	database, err := db.NewPostgresDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize registry database connection
	registryDB, err := db.NewRegistryDB()
	if err != nil {
		log.Fatalf("Failed to connect to registry database: %v", err)
	}
	defer registryDB.Close()

	// Initialize handlers
	serversHandler := handlers.NewServersHandler(registryURL, proxyCache, database)
	ratingsHandler := handlers.NewRatingsHandler(database)
	enhancedHandler := handlers.NewEnhancedHandler(registryDB, database)
	passthroughHandler, err := handlers.NewPassthroughHandler(registryURL, proxyCache)
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

	// Rating endpoints
	mux.HandleFunc("/v0/servers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
		parts := strings.Split(path, "/")

		// Check if the last part of the path is a rating endpoint
		if len(parts) >= 2 {
			lastPart := parts[len(parts)-1]
			switch lastPart {
			case "rate":
				// Write operation - requires authentication
				middleware.APIKeyAuth(ratingsHandler.HandleRate)(w, r)
				return
			case "install":
				// Write operation - requires authentication
				middleware.APIKeyAuth(ratingsHandler.HandleInstall)(w, r)
				return
			case "stats":
				// Read operation - public
				ratingsHandler.HandleStats(w, r)
				return
			case "reviews":
				// Read operation - public
				ratingsHandler.HandleGetReviews(w, r)
				return
			case "feedback":
				// Read operation - public (with pagination)
				ratingsHandler.HandleGetFeedback(w, r)
				return
			}
		}

		// If not a rating endpoint, proxy to upstream
		passthroughHandler.ProxySpecificEndpoint().ServeHTTP(w, r)
	})

	// Publish endpoint (proxy to upstream)
	mux.HandleFunc("/v0/publish", passthroughHandler.ProxySpecificEndpoint())

	// Cache refresh endpoint (our custom endpoint)
	mux.HandleFunc("/v0/cache/refresh", serversHandler.HandleRefresh)

	// Enhanced endpoints (NEW - query registry database directly)
	mux.HandleFunc("/v0/enhanced/servers", enhancedHandler.HandleEnhancedServers)
	mux.HandleFunc("/v0/enhanced/stats/aggregate", enhancedHandler.HandleStats)
	mux.HandleFunc("/v0/enhanced/stats/trending", enhancedHandler.HandleTrending)

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