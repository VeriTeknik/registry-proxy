package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	if limit < 1 || limit > 100 {
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