package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/veriteknik/registry-proxy/internal/cache"
	"github.com/veriteknik/registry-proxy/internal/db"
	"github.com/veriteknik/registry-proxy/internal/handlers"
	_ "github.com/veriteknik/registry-proxy/internal/metrics" // Import metrics for auto-registration
	"github.com/veriteknik/registry-proxy/internal/middleware"
	"github.com/veriteknik/registry-proxy/internal/utils"
	"go.uber.org/zap"
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

	// Initialize metrics IP filter
	if err := middleware.InitMetricsIPFilter(); err != nil {
		log.Fatalf("Failed to initialize metrics IP filter: %v", err)
	}

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
	serversHandler := handlers.NewServersHandler(registryURL, proxyCache, database, registryDB)
	ratingsHandler := handlers.NewRatingsHandler(database, proxyCache)
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

	// Prometheus metrics endpoint (IP-filtered for security)
	mux.Handle("/metrics", middleware.MetricsIPFilter(promhttp.Handler()))

	// Registry-compatible health check
	mux.HandleFunc("/v0/health", func(w http.ResponseWriter, r *http.Request) {
		// Proxy to upstream to get GitHub client ID
		passthroughHandler.ProxySpecificEndpoint().ServeHTTP(w, r)
	})

	// Enriched servers endpoint (only for GET /v0/servers exactly)
	mux.HandleFunc("/v0/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v0/servers" {
			// Use our enriched handler for list
			serversHandler.HandleList(w, r)
		} else if r.URL.Path == "/v0/servers" {
			// Other methods on /v0/servers (POST for publish, etc.) - proxy to upstream
			passthroughHandler.ProxySpecificEndpoint().ServeHTTP(w, r)
		}
		// If path is not exactly "/v0/servers", let it fall through to the "/v0/servers/" handler
	})

	// Server detail and rating endpoints
	mux.HandleFunc("/v0/servers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
		parts := strings.Split(path, "/")

		// Check if it's a user rating endpoint: /servers/{id}/rating/{userId}
		if len(parts) >= 3 && parts[len(parts)-2] == "rating" {
			ratingsHandler.HandleGetUserRating(w, r)
			return
		}

		// Check if the last part of the path is a special endpoint
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

		// If it's a GET request for a server ID (not ending with special endpoint), use our handler
		// Server IDs are like "io.github.user/repo" which has 2 parts when split by "/"
		if r.Method == http.MethodGet && len(parts) <= 2 && parts[0] != "" {
			serversHandler.HandleDetail(w, r)
			return
		}

		// If not a rating endpoint or server detail, proxy to upstream
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

	// Apply middleware stack: timeout -> CORS -> routes
	handler := timeoutMiddleware(corsMiddleware(mux))

	// Start server
	addr := fmt.Sprintf(":%s", port)
	utils.Logger.Info("Starting registry proxy",
		zap.String("addr", addr),
		zap.String("upstream", registryURL),
		zap.Duration("cache_expiration", cacheExpiration),
	)

	if err := http.ListenAndServe(addr, handler); err != nil {
		utils.Logger.Fatal("Server failed", zap.Error(err))
	}

	// Ensure logs are flushed before exit
	defer utils.Sync()
}

// timeoutMiddleware adds request timeout to prevent long-running requests
func timeoutMiddleware(next http.Handler) http.Handler {
	// Get timeout from environment or default to 30 seconds
	timeoutStr := os.Getenv("REQUEST_TIMEOUT")
	timeout := 30 * time.Second
	if timeoutStr != "" {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers
// For public hosted registries, allows all origins by default
func corsMiddleware(next http.Handler) http.Handler {
	// Get allowed origin from environment variable
	// For public hosted registries, "*" is appropriate as the API is designed to be accessed from anywhere
	// Only restrict if you have a specific private deployment scenario
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*" // Public hosted registry - allow all origins
		utils.Logger.Info("CORS_ALLOWED_ORIGIN not set, allowing all origins (public registry mode)",
			zap.String("origin", allowedOrigin))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// For public APIs, also set these headers for better compatibility
		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Credentials", "false")
		}

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}