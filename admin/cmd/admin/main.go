package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/pluggedin/registry-admin/internal/auth"
	"github.com/pluggedin/registry-admin/internal/db"
	"github.com/pluggedin/registry-admin/internal/handlers"
	"github.com/pluggedin/registry-admin/internal/middleware"
)

func main() {
	// Initialize context
	ctx := context.Background()

	// Initialize PostgreSQL connection
	postgresDB, err := db.NewPostgresDB(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	// Initialize operations
	ops := db.NewOperations(postgresDB)

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(jwtManager)
	serversHandler := handlers.NewServersHandler(ops)
	syncHandler := handlers.NewSyncHandler(ops, "https://registry.modelcontextprotocol.io")
	staticHandler := handlers.NewStaticHandler("web/static")

	// Setup router
	router := mux.NewRouter()

	// Apply CORS middleware to all routes
	router.Use(middleware.CORSMiddleware)
	router.Use(middleware.LoggingMiddleware)

	// Public routes
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/health", healthHandler).Methods("GET")

	// Protected API routes
	api := router.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware(jwtManager))

	// Auth endpoints
	api.HandleFunc("/auth/verify", authHandler.Verify).Methods("GET")
	api.HandleFunc("/auth/logout", authHandler.Logout).Methods("POST")

	// Server management endpoints
	api.HandleFunc("/servers", serversHandler.ListServers).Methods("GET")
	api.HandleFunc("/servers", serversHandler.CreateServer).Methods("POST")
	api.HandleFunc("/servers/import", serversHandler.ImportServers).Methods("POST")
	api.HandleFunc("/servers/validate", serversHandler.ValidateServer).Methods("POST")
	api.HandleFunc("/servers/{id}", serversHandler.GetServer).Methods("GET")
	api.HandleFunc("/servers/{id}", serversHandler.UpdateServer).Methods("PUT")
	api.HandleFunc("/servers/{id}", serversHandler.DeleteServer).Methods("DELETE")
	api.HandleFunc("/servers/{id}/status", serversHandler.UpdateStatus).Methods("PATCH")

	// Audit log endpoint
	api.HandleFunc("/audit-logs", serversHandler.GetAuditLogs).Methods("GET")

	// Sync endpoints
	api.HandleFunc("/sync/preview", syncHandler.PreviewSync).Methods("POST")
	api.HandleFunc("/sync/execute", syncHandler.ExecuteSync).Methods("POST")
	api.HandleFunc("/sync/status", syncHandler.GetSyncStatus).Methods("GET")
	api.HandleFunc("/sync/history", syncHandler.GetSyncHistory).Methods("GET")

	// Static files (no auth required for login page)
	router.PathPrefix("/").Handler(staticHandler)

	// Get port from environment
	port := os.Getenv("ADMIN_PORT")
	if port == "" {
		port = "8092"
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Admin server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}