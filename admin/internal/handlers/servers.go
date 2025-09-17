package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pluggedin/registry-admin/internal/db"
	"github.com/pluggedin/registry-admin/internal/middleware"
	"github.com/pluggedin/registry-admin/internal/models"
)

// ServersHandler handles server management endpoints
type ServersHandler struct {
	ops *db.Operations
}

// NewServersHandler creates a new servers handler
func NewServersHandler(ops *db.Operations) *ServersHandler {
	return &ServersHandler{
		ops: ops,
	}
}

// ListServers handles GET /api/servers
func (h *ServersHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 1000 {
		limit = 20
	}

	status := r.URL.Query().Get("status")
	registryName := r.URL.Query().Get("registry_name")
	search := r.URL.Query().Get("search")

	// Get servers from database
	servers, total, err := h.ops.ListServers(r.Context(), page, limit, status, registryName, search)
	if err != nil {
		http.Error(w, "Failed to fetch servers", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"servers": servers,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServer handles GET /api/servers/:id
func (h *ServersHandler) GetServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	server, err := h.ops.GetServer(r.Context(), id)
	if err != nil {
		if err.Error() == "server not found" {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch server", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(server)
}

// CreateServer handles POST /api/servers
func (h *ServersHandler) CreateServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var server models.ServerDetail
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create server
	if err := h.ops.CreateServer(r.Context(), &server); err != nil {
		if err.Error() == "server with name already exists" {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, "Failed to create server", http.StatusInternalServerError)
		}
		return
	}

	// Log audit entry
	user := middleware.GetUserFromContext(r.Context())
	h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
		User:     user,
		Action:   "CREATE_SERVER",
		ServerID: server.ID,
		Details:  "Created server: " + server.Name,
		IP:       r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(server)
}

// UpdateServer handles PUT /api/servers/:id
func (h *ServersHandler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var server models.ServerDetail
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update server
	if err := h.ops.UpdateServer(r.Context(), id, &server); err != nil {
		if err.Error() == "server not found" {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update server", http.StatusInternalServerError)
		}
		return
	}

	// Log audit entry
	user := middleware.GetUserFromContext(r.Context())
	h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
		User:     user,
		Action:   "UPDATE_SERVER",
		ServerID: id,
		Details:  "Updated server: " + server.Name,
		IP:       r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(server)
}

// DeleteServer handles DELETE /api/servers/:id
func (h *ServersHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Get server details before deletion for audit log
	server, _ := h.ops.GetServer(r.Context(), id)
	serverName := "Unknown"
	if server != nil {
		serverName = server.Name
	}

	// Delete server
	if err := h.ops.DeleteServer(r.Context(), id); err != nil {
		if err.Error() == "server not found" {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete server", http.StatusInternalServerError)
		}
		return
	}

	// Log audit entry
	user := middleware.GetUserFromContext(r.Context())
	h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
		User:     user,
		Action:   "DELETE_SERVER",
		ServerID: id,
		Details:  "Deleted server: " + serverName,
		IP:       r.RemoteAddr,
	})

	w.WriteHeader(http.StatusNoContent)
}

// UpdateStatus handles PATCH /api/servers/:id/status
func (h *ServersHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	status := models.ServerStatus(req.Status)
	if status != models.ServerStatusActive && status != models.ServerStatusDeprecated {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	// Update status
	if err := h.ops.UpdateStatus(r.Context(), id, status); err != nil {
		if err.Error() == "server not found" {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
		}
		return
	}

	// Log audit entry
	user := middleware.GetUserFromContext(r.Context())
	h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
		User:     user,
		Action:   "UPDATE_STATUS",
		ServerID: id,
		Details:  "Updated status to: " + req.Status,
		IP:       r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": req.Status})
}

// GetAuditLogs handles GET /api/audit-logs
func (h *ServersHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}

	logs, err := h.ops.GetAuditLogs(r.Context(), limit)
	if err != nil {
		http.Error(w, "Failed to fetch audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// ImportServers handles POST /api/servers/import
func (h *ServersHandler) ImportServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := models.ImportResponse{
		Success: []models.ImportResult{},
		Failed:  []models.ImportResult{},
	}

	user := middleware.GetUserFromContext(r.Context())

	for _, server := range req.Servers {
		// Check if server exists
		if !req.Options.UpdateExisting {
			exists, _ := h.ops.ServerExists(r.Context(), server.Name)
			if exists {
				if req.Options.SkipExisting {
					continue
				}
				response.Failed = append(response.Failed, models.ImportResult{
					Name:  server.Name,
					Error: "Server already exists",
				})
				continue
			}
		}

		// Generate ID if not provided
		if server.ID == "" {
			server.ID = uuid.New().String()
		}

		// Create or update server
		if req.Options.UpdateExisting {
			if err := h.ops.UpdateServer(r.Context(), server.ID, &server); err != nil {
				response.Failed = append(response.Failed, models.ImportResult{
					Name:  server.Name,
					Error: err.Error(),
				})
			} else {
				response.Success = append(response.Success, models.ImportResult{
					Name: server.Name,
					ID:   server.ID,
				})
				
				// Log audit entry
				h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
					User:     user,
					Action:   "UPDATE_SERVER",
					ServerID: server.ID,
					Details:  "Updated server via import: " + server.Name,
					IP:       r.RemoteAddr,
				})
			}
		} else {
			if err := h.ops.CreateServer(r.Context(), &server); err != nil {
				response.Failed = append(response.Failed, models.ImportResult{
					Name:  server.Name,
					Error: err.Error(),
				})
			} else {
				response.Success = append(response.Success, models.ImportResult{
					Name: server.Name,
					ID:   server.ID,
				})
				
				// Log audit entry
				h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
					User:     user,
					Action:   "CREATE_SERVER",
					ServerID: server.ID,
					Details:  "Created server via import: " + server.Name,
					IP:       r.RemoteAddr,
				})
			}
		}
	}

	response.Summary = models.ImportSummary{
		Total:   len(req.Servers),
		Success: len(response.Success),
		Failed:  len(response.Failed),
	}

	// Log summary audit entry
	h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
		User:    user,
		Action:  "BATCH_IMPORT",
		Details: fmt.Sprintf("Imported %d/%d servers successfully", response.Summary.Success, response.Summary.Total),
		IP:      r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ValidateServer handles POST /api/servers/validate
func (h *ServersHandler) ValidateServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var server models.ServerDetail
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := models.ValidationResponse{Valid: true}
	
	// Basic validation
	var errors []string
	
	// Required fields
	if server.Name == "" {
		errors = append(errors, "Name is required")
	}
	if server.Description == "" {
		errors = append(errors, "Description is required")
	}
	if server.VersionDetail.Version == "" {
		errors = append(errors, "Version is required")
	}
	
	// Repository validation
	if server.Repository.URL != "" {
		if server.Repository.Source != "github" {
			errors = append(errors, "Repository source must be 'github'")
		}
		if !strings.Contains(server.Repository.URL, "github.com") {
			errors = append(errors, "Repository URL must be a GitHub URL")
		}
	}
	
	// Package validation
	for i, pkg := range server.Packages {
		// Check registry name
		validRegistries := map[string]bool{
			"npm":    true,
			"pypi":   true,
			"docker": true,
			"nuget":  true,
		}
		if !validRegistries[pkg.RegistryName] {
			errors = append(errors, fmt.Sprintf("Package %d: Invalid registry name '%s'", i, pkg.RegistryName))
		}
		
		// Check runtime hint
		if pkg.RuntimeHint != "" {
			validHints := map[string]bool{
				"npx":    true,
				"uvx":    true,
				"docker": true,
				"dnx":    true,
			}
			if !validHints[pkg.RuntimeHint] {
				errors = append(errors, fmt.Sprintf("Package %d: Invalid runtime hint '%s'", i, pkg.RuntimeHint))
			}
		}
		
		if pkg.Name == "" {
			errors = append(errors, fmt.Sprintf("Package %d: Name is required", i))
		}
		if pkg.Version == "" {
			errors = append(errors, fmt.Sprintf("Package %d: Version is required", i))
		}
	}
	
	if len(errors) > 0 {
		response.Valid = false
		response.Errors = errors
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}