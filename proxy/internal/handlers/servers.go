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
	registryDB     *db.DB
}

// NewServersHandler creates a new servers handler
func NewServersHandler(registryURL string, cache *cache.Cache, database *db.DB, registryDB *db.DB) *ServersHandler {
	return &ServersHandler{
		registryClient: client.NewRegistryClient(registryURL),
		cache:          cache,
		db:             database,
		registryDB:     registryDB,
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
	// Support both registry_name and packageRegistry parameters
	if registryName == "" {
		registryName = r.URL.Query().Get("packageRegistry")
	}
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

// getEnrichedServers fetches enriched servers from cache or database
func (h *ServersHandler) getEnrichedServers(ctx context.Context) ([]models.EnrichedServer, error) {
	// Try cache first
	if servers, found := h.cache.GetServers(); found {
		return servers, nil
	}

	// Fetch directly from registry database (includes all fields)
	filter := db.ServerFilter{}
	serverMaps, _, err := h.registryDB.QueryServersEnhanced(ctx, filter, "published_at", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("fetching servers from database: %w", err)
	}

	// Convert to EnrichedServer format
	servers := make([]models.EnrichedServer, 0, len(serverMaps))
	for _, serverMap := range serverMaps {
		enriched := h.convertMapToEnrichedServer(serverMap)
		servers = append(servers, enriched)
	}

	// Cache the result
	h.cache.SetServers(servers)

	return servers, nil
}

// convertMapToEnrichedServer converts a database map to EnrichedServer
func (h *ServersHandler) convertMapToEnrichedServer(serverMap map[string]interface{}) models.EnrichedServer {
	enriched := models.EnrichedServer{}

	// Extract basic fields
	if id, ok := serverMap["id"].(string); ok {
		enriched.ID = id
		enriched.Name = id // Use ID as name
	}

	if desc, ok := serverMap["description"].(string); ok {
		enriched.Description = desc
	}

	// Extract repository
	if repo, ok := serverMap["repository"].(map[string]interface{}); ok {
		enriched.Repository = models.Repository{
			URL:    getStringFromMap(repo, "url"),
			Source: getStringFromMap(repo, "source"),
			ID:     getStringFromMap(repo, "id"),
		}
	}

	// Extract version detail
	if vd, ok := serverMap["version_detail"].(map[string]interface{}); ok {
		enriched.VersionDetail = models.VersionDetail{
			Version:     getStringFromMap(vd, "version"),
			ReleaseDate: getStringFromMap(vd, "release_date"),
			IsLatest:    getBoolFromMap(vd, "is_latest"),
		}
	}

	// Extract packages (IMPORTANT: this includes packageArguments and runtimeArguments)
	if packages, ok := serverMap["packages"].([]interface{}); ok {
		for _, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]interface{}); ok {
				enriched.Packages = append(enriched.Packages, h.convertPackage(pkgMap))
			}
		}
	}

	// Extract remotes
	if remotes, ok := serverMap["remotes"].([]interface{}); ok {
		for _, remote := range remotes {
			if remoteMap, ok := remote.(map[string]interface{}); ok {
				enriched.Remotes = append(enriched.Remotes, models.Remote{
					TransportType: getStringFromMap(remoteMap, "type"),
					URL:           getStringFromMap(remoteMap, "url"),
				})
			}
		}
	}

	// Extract stats
	if stats, ok := serverMap["stats"].(map[string]interface{}); ok {
		if rating, ok := stats["rating"].(float64); ok {
			enriched.Rating = rating
		}
		if ratingCount, ok := stats["rating_count"].(int); ok {
			enriched.RatingCount = ratingCount
		}
		if installCount, ok := stats["install_count"].(int); ok {
			enriched.InstallationCount = installCount
		}
	}

	return enriched
}

// convertPackage converts a package map to models.Package
func (h *ServersHandler) convertPackage(pkgMap map[string]interface{}) models.Package {
	pkg := models.Package{
		RegistryName: getStringFromMap(pkgMap, "registryType"),
		Name:         getStringFromMap(pkgMap, "identifier"),
		Version:      getStringFromMap(pkgMap, "version"),
		RuntimeHint:  getStringFromMap(pkgMap, "runtimeHint"),
	}

	// Extract transport
	if transport, ok := pkgMap["transport"].(map[string]interface{}); ok {
		pkg.Transport = &models.Transport{
			Type: getStringFromMap(transport, "type"),
		}
	}

	// Extract environment variables
	if envVars, ok := pkgMap["environmentVariables"].([]interface{}); ok {
		for _, ev := range envVars {
			if evMap, ok := ev.(map[string]interface{}); ok {
				pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, models.EnvironmentVariable{
					Name:        getStringFromMap(evMap, "name"),
					Description: getStringFromMap(evMap, "description"),
					Default:     getStringFromMap(evMap, "default"),
					IsRequired:  getBoolFromMap(evMap, "isRequired"),
					IsSecret:    getBoolFromMap(evMap, "isSecret"),
				})
			}
		}
	}

	// Extract runtime arguments (CRITICAL)
	if runtimeArgs, ok := pkgMap["runtimeArguments"].([]interface{}); ok {
		for _, arg := range runtimeArgs {
			if argMap, ok := arg.(map[string]interface{}); ok {
				pkg.RuntimeArguments = append(pkg.RuntimeArguments, h.convertArgument(argMap))
			}
		}
	}

	// Extract package arguments (CRITICAL)
	if packageArgs, ok := pkgMap["packageArguments"].([]interface{}); ok {
		for _, arg := range packageArgs {
			if argMap, ok := arg.(map[string]interface{}); ok {
				pkg.PackageArguments = append(pkg.PackageArguments, h.convertArgument(argMap))
			}
		}
	}

	return pkg
}

// convertArgument converts an argument map to models.Argument
func (h *ServersHandler) convertArgument(argMap map[string]interface{}) models.Argument {
	arg := models.Argument{
		Type:        getStringFromMap(argMap, "type"),
		Name:        getStringFromMap(argMap, "name"),
		Value:       getStringFromMap(argMap, "value"),
		Default:     getStringFromMap(argMap, "default"),
		Description: getStringFromMap(argMap, "description"),
		IsRequired:  getBoolFromMap(argMap, "isRequired"),
	}

	// Extract choices if present
	if choices, ok := argMap["choices"].([]interface{}); ok {
		for _, choice := range choices {
			if choiceStr, ok := choice.(string); ok {
				arg.Choices = append(arg.Choices, choiceStr)
			}
		}
	}

	return arg
}

// Helper functions to safely extract values from maps
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
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
// Supports comma-separated values like "npm,pypi"
func (h *ServersHandler) filterByRegistryName(servers []models.EnrichedServer, registryName string) []models.EnrichedServer {
	if registryName == "" {
		return servers
	}

	// Split by comma to support multiple registry names
	registryNames := strings.Split(registryName, ",")
	for i := range registryNames {
		registryNames[i] = strings.TrimSpace(registryNames[i])
	}

	filtered := make([]models.EnrichedServer, 0)
	for _, server := range servers {
		for _, pkg := range server.Packages {
			// Check if package registry matches any of the requested registries
			for _, rName := range registryNames {
				if strings.EqualFold(pkg.RegistryName, rName) {
					filtered = append(filtered, server)
					goto nextServer
				}
			}
		}
		nextServer:
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
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Cache refreshed successfully",
		"updated_at": h.cache.GetLastUpdate(),
	}); err != nil {
		log.Printf("Error encoding refresh response: %v", err)
	}
}