package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/veriteknik/registry-proxy/internal/cache"
	"github.com/veriteknik/registry-proxy/internal/client"
	"github.com/veriteknik/registry-proxy/internal/db"
	"github.com/veriteknik/registry-proxy/internal/models"
)

// ServersHandler handles enriched server list requests
type ServersHandler struct {
	registryClient *client.RegistryClient
	cache          *cache.Cache
	db             *db.DB
}

// NewServersHandler creates a new servers handler
func NewServersHandler(registryURL string, cache *cache.Cache, database *db.DB) *ServersHandler {
	return &ServersHandler{
		registryClient: client.NewRegistryClient(registryURL),
		cache:          cache,
		db:             database,
	}
}

// HandleList handles GET /v0/servers with enriched data
func (h *ServersHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	registryName := r.URL.Query().Get("registry_name")
	sortBy := r.URL.Query().Get("sort")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	search := strings.ToLower(r.URL.Query().Get("search"))

	limit := 30
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 500 {
				limit = 500
			}
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get enriched servers (from cache or fetch)
	servers, err := h.getEnrichedServers(r.Context())
	if err != nil {
		log.Printf("Error getting enriched servers: %v", err)
		http.Error(w, "Failed to fetch servers", http.StatusInternalServerError)
		return
	}

	// Apply search filter
	if search != "" {
		servers = h.filterBySearch(servers, search)
	}

	// Apply registry_name filter
	if registryName != "" {
		servers = h.filterByRegistryName(servers, registryName)
	}

	// Apply sorting
	servers = h.sortServers(servers, sortBy)

	// Apply pagination
	total := len(servers)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paginatedServers := servers[start:end]

	// Prepare response
	response := models.ProxyResponse{
		Servers: paginatedServers,
		Metadata: models.ResponseMetadata{
			Count:      len(paginatedServers),
			Total:      total,
			FilteredBy: registryName,
			SortedBy:   sortBy,
			CachedAt:   h.cache.GetLastUpdate(),
		},
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	
	// Encode response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// getEnrichedServers fetches enriched servers from cache or upstream
func (h *ServersHandler) getEnrichedServers(ctx context.Context) ([]models.EnrichedServer, error) {
	// Try cache first
	if servers, found := h.cache.GetServers(); found {
		return servers, nil
	}

	// Fetch from upstream
	servers, err := h.registryClient.GetAllServersWithDetails(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching servers: %w", err)
	}

	// Enrich with stats from database
	if h.db != nil {
		for i := range servers {
			rating, ratingCount, installCount, err := h.db.GetServerStats(ctx, servers[i].ID)
			if err != nil {
				log.Printf("Warning: Failed to get stats for server %s: %v", servers[i].ID, err)
				// Continue without stats rather than failing the entire request
				continue
			}
			servers[i].Rating = rating
			servers[i].RatingCount = ratingCount
			servers[i].InstallationCount = installCount
		}
	}

	// Cache the result
	h.cache.SetServers(servers)

	return servers, nil
}

// filterBySearch filters servers by search term in name or description
func (h *ServersHandler) filterBySearch(servers []models.EnrichedServer, search string) []models.EnrichedServer {
	if search == "" {
		return servers
	}

	filtered := make([]models.EnrichedServer, 0)
	for _, server := range servers {
		if strings.Contains(strings.ToLower(server.Name), search) ||
			strings.Contains(strings.ToLower(server.Description), search) {
			filtered = append(filtered, server)
		}
	}
	return filtered
}

// filterByRegistryName filters servers by package registry name
func (h *ServersHandler) filterByRegistryName(servers []models.EnrichedServer, registryName string) []models.EnrichedServer {
	if registryName == "" {
		return servers
	}

	filtered := make([]models.EnrichedServer, 0)
	for _, server := range servers {
		for _, pkg := range server.Packages {
			if strings.EqualFold(pkg.RegistryName, registryName) {
				filtered = append(filtered, server)
				break
			}
		}
	}
	return filtered
}

// sortServers sorts servers based on the sort parameter
func (h *ServersHandler) sortServers(servers []models.EnrichedServer, sortBy string) []models.EnrichedServer {
	// Create a copy to avoid modifying the original
	sorted := make([]models.EnrichedServer, len(servers))
	copy(sorted, servers)

	switch sortBy {
	case "date_asc", "release_date_asc":
		sort.Slice(sorted, func(i, j int) bool {
			ti, _ := time.Parse(time.RFC3339, sorted[i].VersionDetail.ReleaseDate)
			tj, _ := time.Parse(time.RFC3339, sorted[j].VersionDetail.ReleaseDate)
			return ti.Before(tj)
		})
	case "date_desc", "release_date_desc", "newest":
		sort.Slice(sorted, func(i, j int) bool {
			ti, _ := time.Parse(time.RFC3339, sorted[i].VersionDetail.ReleaseDate)
			tj, _ := time.Parse(time.RFC3339, sorted[j].VersionDetail.ReleaseDate)
			return ti.After(tj)
		})
	case "name_asc", "alphabetical":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name < sorted[j].Name
		})
	case "name_desc":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name > sorted[j].Name
		})
	default:
		// Default: newest first
		sort.Slice(sorted, func(i, j int) bool {
			ti, _ := time.Parse(time.RFC3339, sorted[i].VersionDetail.ReleaseDate)
			tj, _ := time.Parse(time.RFC3339, sorted[j].VersionDetail.ReleaseDate)
			return ti.After(tj)
		})
	}

	return sorted
}

// HandleRefresh forces a cache refresh
func (h *ServersHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear cache
	h.cache.Clear()

	// Fetch fresh data
	ctx := r.Context()
	_, err := h.getEnrichedServers(ctx)
	if err != nil {
		log.Printf("Error refreshing cache: %v", err)
		http.Error(w, "Failed to refresh cache", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Cache refreshed successfully",
		"updated_at": h.cache.GetLastUpdate(),
	})
}