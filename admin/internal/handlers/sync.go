package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pluggedin/registry-admin/internal/db"
	"github.com/pluggedin/registry-admin/internal/middleware"
	"github.com/pluggedin/registry-admin/internal/models"
)

// SyncHandler handles registry synchronization endpoints
type SyncHandler struct {
	ops                 *db.Operations
	officialRegistryURL string
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(ops *db.Operations, officialRegistryURL string) *SyncHandler {
	if officialRegistryURL == "" {
		officialRegistryURL = "https://registry.modelcontextprotocol.io"
	}
	return &SyncHandler{
		ops:                 ops,
		officialRegistryURL: officialRegistryURL,
	}
}

// SyncRequest represents a sync request
type SyncRequest struct {
	DryRun         bool `json:"dry_run"`
	UpdateExisting bool `json:"update_existing"`
	AddNew         bool `json:"add_new"`
}

// SyncResult represents sync results
type SyncResult struct {
	NewServers []ServerSummary `json:"new_servers"`
	Updates    []UpdateSummary `json:"updates"`
	Unchanged  int             `json:"unchanged"`
	Errors     []string        `json:"errors,omitempty"`
	Added      int             `json:"added"`
	Updated    int             `json:"updated"`
}

// ServerSummary represents a server summary
type ServerSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	RepoSource  string   `json:"repo_source"`
	Types       []string `json:"types"` // Package registry types or remote types
}

// UpdateSummary represents an update summary
type UpdateSummary struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
}

// OfficialRegistryResponse represents the official registry API response
type OfficialRegistryResponse struct {
	Servers  []OfficialServerItem `json:"servers"`
	Metadata RegistryMetadata     `json:"metadata"`
}

// RegistryMetadata represents pagination metadata
type RegistryMetadata struct {
	NextCursor string `json:"nextCursor"`
	Count      int    `json:"count"`
}

// OfficialServerItem represents a single server item from the official registry
type OfficialServerItem struct {
	Server OfficialServer `json:"server"`
	Meta   OfficialMeta   `json:"_meta"`
}

// OfficialServer represents a server from the official registry (with different field names)
type OfficialServer struct {
	Schema      string                   `json:"$schema"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Repository  models.Repository        `json:"repository"`
	Version     string                   `json:"version"`
	Packages    []OfficialPackage        `json:"packages,omitempty"`
	Remotes     []OfficialRemote         `json:"remotes,omitempty"`
}

// OfficialPackage represents a package from the official registry
type OfficialPackage struct {
	RegistryType         string                        `json:"registryType"`
	Identifier           string                        `json:"identifier"`
	Transport            map[string]interface{}        `json:"transport,omitempty"`
	EnvironmentVariables []models.EnvironmentVariable  `json:"environmentVariables,omitempty"`
}

// OfficialRemote represents a remote from the official registry
type OfficialRemote struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// OfficialMeta represents the metadata from the official registry
type OfficialMeta struct {
	Official OfficialMetadata `json:"io.modelcontextprotocol.registry/official"`
}

// OfficialMetadata represents the official registry metadata
type OfficialMetadata struct {
	Status      string `json:"status"`
	PublishedAt string `json:"publishedAt"`
	UpdatedAt   string `json:"updatedAt"`
	IsLatest    bool   `json:"isLatest"`
}

// PreviewSync handles POST /api/sync/preview
func (h *SyncHandler) PreviewSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Always run preview in dry-run mode
	req.DryRun = true

	result, err := h.performSync(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Sync preview failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ExecuteSync handles POST /api/sync/execute
func (h *SyncHandler) ExecuteSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Force execute mode
	req.DryRun = false

	result, err := h.performSync(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Sync execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit entry
	user := middleware.GetUserFromContext(r.Context())
	h.ops.LogAuditEntry(r.Context(), &models.AuditLog{
		User:    user,
		Action:  "SYNC_REGISTRY",
		Details: fmt.Sprintf("Synced with official registry. Added: %d, Updated: %d", result.Added, result.Updated),
		IP:      r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// performSync performs the actual sync operation
func (h *SyncHandler) performSync(ctx context.Context, req SyncRequest) (*SyncResult, error) {
	result := &SyncResult{
		NewServers: []ServerSummary{},
		Updates:    []UpdateSummary{},
		Errors:     []string{},
	}

	// Fetch servers from official registry
	officialServers, err := h.fetchOfficialServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching official servers: %w", err)
	}

	// Get existing servers from database
	existingServers, _, err := h.ops.ListServers(ctx, 1, 1000, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("fetching existing servers: %w", err)
	}

	// Create map of existing servers by name for quick lookup
	existingMap := make(map[string]*models.ServerDetail)
	for i := range existingServers {
		existingMap[existingServers[i].Name] = &existingServers[i]
	}

	// Process each official server (already filtered to latest versions only)
	for _, officialServer := range officialServers {
		existing, exists := existingMap[officialServer.Name]

		if !exists {
			// New server
			if req.AddNew {
				result.NewServers = append(result.NewServers, ServerSummary{
					Name:        officialServer.Name,
					Description: officialServer.Description,
					Version:     officialServer.VersionDetail.Version,
					RepoSource:  officialServer.Repository.Source,
					Types:       extractServerTypes(&officialServer),
				})

				if !req.DryRun {
					// Add source metadata
					officialServer.Status = models.ServerStatusActive

					// Create the server
					if err := h.ops.CreateServer(ctx, &officialServer); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("Failed to add %s: %v", officialServer.Name, err))
					} else {
						result.Added++
					}
				}
			}
		} else {
			// Existing server - check if update needed
			if req.UpdateExisting && h.shouldUpdate(existing, &officialServer) {
				result.Updates = append(result.Updates, UpdateSummary{
					Name:           officialServer.Name,
					CurrentVersion: existing.VersionDetail.Version,
					NewVersion:     officialServer.VersionDetail.Version,
				})

				if !req.DryRun {
					// Preserve status if it exists
					if existing.Status != "" {
						officialServer.Status = existing.Status
					}

					// Update the server
					if err := h.ops.UpdateServer(ctx, existing.ID, &officialServer); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("Failed to update %s: %v", officialServer.Name, err))
					} else {
						result.Updated++
					}
				}
			} else {
				result.Unchanged++
			}
		}
	}

	return result, nil
}

// fetchOfficialServers fetches servers from the official registry with pagination
func (h *SyncHandler) fetchOfficialServers(ctx context.Context) ([]models.ServerDetail, error) {
	const maxPages = 100 // Safety limit to prevent infinite loops
	allServers := make([]models.ServerDetail, 0)
	cursor := ""
	pageCount := 0
	client := &http.Client{Timeout: 30 * time.Second}

	for {
		pageCount++
		if pageCount > maxPages {
			return nil, fmt.Errorf("pagination limit exceeded after %d pages: possible infinite loop or rate limiting issue", maxPages)
		}

		// Build URL with pagination
		url := fmt.Sprintf("%s/v0/servers?limit=100", h.officialRegistryURL)
		if cursor != "" {
			url = fmt.Sprintf("%s&cursor=%s", url, cursor)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var registryResp OfficialRegistryResponse
		if err := json.NewDecoder(resp.Body).Decode(&registryResp); err != nil {
			return nil, err
		}

		// Extract server details from nested structure, filtering for latest versions only
		for _, item := range registryResp.Servers {
			// Only include servers marked as latest version
			if item.Meta.Official.IsLatest {
				serverDetail := convertOfficialToServerDetail(&item.Server)
				allServers = append(allServers, serverDetail)
			}
		}

		// Check if there are more pages
		if registryResp.Metadata.NextCursor == "" {
			break
		}

		cursor = registryResp.Metadata.NextCursor
	}

	return allServers, nil
}

// convertOfficialToServerDetail converts an OfficialServer to ServerDetail
func convertOfficialToServerDetail(official *OfficialServer) models.ServerDetail {
	server := models.ServerDetail{
		Server: models.Server{
			Name:        official.Name,
			Description: official.Description,
			Repository:  official.Repository,
			VersionDetail: models.VersionDetail{
				Version:  official.Version,
				IsLatest: true,
			},
		},
	}

	// Convert packages
	for _, officialPkg := range official.Packages {
		server.Packages = append(server.Packages, models.Package{
			RegistryName:         officialPkg.RegistryType,
			Name:                 officialPkg.Identifier,
			EnvironmentVariables: officialPkg.EnvironmentVariables,
		})
	}

	// Convert remotes
	for _, officialRemote := range official.Remotes {
		server.Remotes = append(server.Remotes, models.Remote{
			TransportType: officialRemote.Type,
			URL:           officialRemote.URL,
		})
	}

	return server
}

// mapRegistryTypeToFriendlyName converts technical registry types to user-friendly names
func mapRegistryTypeToFriendlyName(registryType string) string {
	mapping := map[string]string{
		"oci":              "docker",
		"pypi":             "python",
		"npm":              "npm",
		"sse":              "remote",
		"streamable-http":  "remote",
	}

	if friendly, ok := mapping[registryType]; ok {
		return friendly
	}
	return registryType
}

// extractServerTypes extracts package registry types or remote types from a server
func extractServerTypes(server *models.ServerDetail) []string {
	types := []string{}

	// Add package registry types
	for _, pkg := range server.Packages {
		if pkg.RegistryName != "" {
			friendlyName := mapRegistryTypeToFriendlyName(pkg.RegistryName)
			types = append(types, friendlyName)
		}
	}

	// Add remote types if no packages
	if len(types) == 0 {
		for _, remote := range server.Remotes {
			if remote.TransportType != "" {
				friendlyName := mapRegistryTypeToFriendlyName(remote.TransportType)
				types = append(types, friendlyName)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, t := range types {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}

	return unique
}

// shouldUpdate determines if a server should be updated
func (h *SyncHandler) shouldUpdate(existing *models.ServerDetail, official *models.ServerDetail) bool {
	// Update if version is different
	if existing.VersionDetail.Version != official.VersionDetail.Version {
		return true
	}

	// Update if release date is newer
	existingDate, _ := time.Parse(time.RFC3339, existing.VersionDetail.ReleaseDate)
	officialDate, _ := time.Parse(time.RFC3339, official.VersionDetail.ReleaseDate)
	if officialDate.After(existingDate) {
		return true
	}

	// Update if packages have changed
	if len(existing.Packages) != len(official.Packages) {
		return true
	}

	// Check for package differences
	for i, pkg := range official.Packages {
		if i >= len(existing.Packages) {
			return true
		}
		if existing.Packages[i].Name != pkg.Name ||
		   existing.Packages[i].Version != pkg.Version {
			return true
		}
	}

	// Check if remotes field exists in official but not in existing
	if len(official.Remotes) > 0 && len(existing.Remotes) == 0 {
		return true
	}

	return false
}

// GetSyncStatus handles GET /api/sync/status
func (h *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get last sync from audit logs
	logs, err := h.ops.GetAuditLogs(r.Context(), 100)
	if err != nil {
		http.Error(w, "Failed to fetch audit logs", http.StatusInternalServerError)
		return
	}

	var lastSync *models.AuditLog
	for _, log := range logs {
		if log.Action == "SYNC_REGISTRY" {
			lastSync = &log
			break
		}
	}

	status := map[string]interface{}{
		"last_sync": nil,
		"official_registry_url": h.officialRegistryURL,
	}

	if lastSync != nil {
		status["last_sync"] = map[string]interface{}{
			"timestamp": lastSync.Timestamp,
			"user":      lastSync.User,
			"details":   lastSync.Details,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetSyncHistory handles GET /api/sync/history
func (h *SyncHandler) GetSyncHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get sync-related audit logs
	logs, err := h.ops.GetAuditLogs(r.Context(), 100)
	if err != nil {
		http.Error(w, "Failed to fetch audit logs", http.StatusInternalServerError)
		return
	}

	syncHistory := []models.AuditLog{}
	for _, log := range logs {
		if strings.Contains(log.Action, "SYNC") {
			syncHistory = append(syncHistory, log)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncHistory)
}