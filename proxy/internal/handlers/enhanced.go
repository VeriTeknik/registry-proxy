package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/veriteknik/registry-proxy/internal/db"
)

// EnhancedHandler handles enhanced endpoints that query the registry database directly
type EnhancedHandler struct {
	registryDB *db.DB
	proxyDB    *db.DB
}

// NewEnhancedHandler creates a new enhanced handler
func NewEnhancedHandler(registryDB, proxyDB *db.DB) *EnhancedHandler {
	return &EnhancedHandler{
		registryDB: registryDB,
		proxyDB:    proxyDB,
	}
}

// HandleEnhancedServers handles /v0/enhanced/servers with filtering and sorting
func (h *EnhancedHandler) HandleEnhancedServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Create filter from query parameters
	filter := db.ServerFilter{
		Search:       query.Get("search"),
		Category:     query.Get("category"),
		MinRating:    parseFloat(query.Get("min_rating")),
		MinInstalls:  parseInt(query.Get("min_installs")),
	}

	// Parse registry types (including special "remote" handling)
	if registryTypes := query.Get("registry_types"); registryTypes != "" {
		filter.RegistryTypes = strings.Split(registryTypes, ",")
	}

	// Parse tags
	if tags := query.Get("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
	}

	// Parse transport types
	if transports := query.Get("transports"); transports != "" {
		filter.HasTransport = strings.Split(transports, ",")
	}

	// Pagination
	limit := parseInt(query.Get("limit"))
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit for safety
	}

	offset := parseInt(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Sorting
	sort := query.Get("sort")
	if sort == "" {
		sort = "created"
	}

	// Query enhanced servers
	servers, totalCount, err := h.registryDB.QueryServersEnhanced(r.Context(), filter, sort, limit, offset)
	if err != nil {
		log.Printf("Error querying enhanced servers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"servers":     servers,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
		"filters":     filter,
		"sort":        sort,
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))

	// Write response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleStats handles /v0/enhanced/stats/aggregate
func (h *EnhancedHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Query database for stats
	query := `
		SELECT
			COUNT(DISTINCT s.server_name) as total_servers,
			COUNT(DISTINCT CASE WHEN jsonb_array_length(s.value->'packages') > 0 THEN s.server_name END) as servers_with_packages,
			COUNT(DISTINCT ss.server_id) as rated_servers,
			COALESCE(AVG(ss.rating), 0) as avg_rating,
			COALESCE(SUM(ss.rating_count), 0) as total_reviews,
			COALESCE(SUM(ss.installation_count), 0) as total_installs,
			COUNT(DISTINCT CASE
				WHEN s.value->'packages' @> '[{"registry_name":"npm"}]' THEN s.server_name
			END) as npm_count,
			COUNT(DISTINCT CASE
				WHEN s.value->'packages' @> '[{"registry_name":"pypi"}]' THEN s.server_name
			END) as pypi_count,
			COUNT(DISTINCT CASE
				WHEN s.value->'packages' @> '[{"registry_name":"oci"}]' THEN s.server_name
			END) as oci_count,
			COUNT(DISTINCT CASE
				WHEN EXISTS (
					SELECT 1 FROM jsonb_array_elements(s.value->'remotes') r
					WHERE r->>'transport_type' IN ('sse', 'http', 'streamable-http')
				) THEN s.server_name
			END) as remote_count,
			COUNT(DISTINCT CASE
				WHEN s.published_at > NOW() - INTERVAL '7 days' THEN s.server_name
			END) as new_this_week,
			COUNT(DISTINCT CASE
				WHEN s.updated_at > NOW() - INTERVAL '7 days' THEN s.server_name
			END) as updated_this_week
		FROM servers s
		LEFT JOIN proxy_server_stats ss ON s.server_name = ss.server_id
		WHERE s.is_latest = true
	`

	var stats struct {
		TotalServers        int            `json:"total_servers"`
		ServersWithPackages int            `json:"servers_with_packages"`
		RatedServers        int            `json:"rated_servers"`
		AvgRating           float64        `json:"average_rating"`
		TotalReviews        int            `json:"total_reviews"`
		TotalInstalls       int            `json:"total_installs"`
		NPMCount            int            `json:"npm_count"`
		PyPICount           int            `json:"pypi_count"`
		OCICount            int            `json:"oci_count"`
		RemoteCount         int            `json:"remote_count"`
		NewThisWeek         int            `json:"new_this_week"`
		UpdatedThisWeek     int            `json:"updated_this_week"`
		RegistryBreakdown   map[string]int `json:"registry_breakdown"`
	}

	err := h.registryDB.QueryRowContext(r.Context(), query).Scan(
		&stats.TotalServers,
		&stats.ServersWithPackages,
		&stats.RatedServers,
		&stats.AvgRating,
		&stats.TotalReviews,
		&stats.TotalInstalls,
		&stats.NPMCount,
		&stats.PyPICount,
		&stats.OCICount,
		&stats.RemoteCount,
		&stats.NewThisWeek,
		&stats.UpdatedThisWeek,
	)

	if err != nil {
		log.Printf("Error querying stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Add registry breakdown
	stats.RegistryBreakdown = map[string]int{
		"npm":    stats.NPMCount,
		"pypi":   stats.PyPICount,
		"oci":    stats.OCICount,
		"remote": stats.RemoteCount,
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")

	// Write response
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleTrending handles /v0/enhanced/stats/trending
func (h *EnhancedHandler) HandleTrending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get trending servers (high activity in last 7 days)
	query := `
		WITH trending AS (
			SELECT
				s.server_name,
				s.value,
				COALESCE(ss.rating, 0) as rating,
				COALESCE(ss.rating_count, 0) as rating_count,
				COALESCE(ss.installation_count, 0) as installation_count,
				(
					COALESCE(ss.installation_count, 0) * 0.3 +
					COALESCE(ss.rating_count, 0) * 0.3 +
					COALESCE(ss.rating, 0) * 10 +
					CASE
						WHEN s.updated_at > NOW() - INTERVAL '7 days' THEN 20
						WHEN s.updated_at > NOW() - INTERVAL '30 days' THEN 10
						ELSE 0
					END
				) as trending_score
			FROM servers s
			LEFT JOIN proxy_server_stats ss ON s.server_name = ss.server_id
			WHERE s.is_latest = true
			AND (
				s.published_at > NOW() - INTERVAL '30 days'
				OR s.updated_at > NOW() - INTERVAL '30 days'
				OR ss.updated_at > NOW() - INTERVAL '30 days'
			)
		)
		SELECT
			server_name,
			value,
			rating,
			rating_count,
			installation_count,
			trending_score
		FROM trending
		ORDER BY trending_score DESC
		LIMIT 10
	`

	rows, err := h.registryDB.QueryContext(r.Context(), query)
	if err != nil {
		log.Printf("Error querying trending: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	trending := []map[string]interface{}{}
	for rows.Next() {
		var serverName string
		var valueJSON []byte
		var rating, trendingScore float64
		var ratingCount, installCount int

		err := rows.Scan(
			&serverName,
			&valueJSON,
			&rating,
			&ratingCount,
			&installCount,
			&trendingScore,
		)
		if err != nil {
			log.Printf("Error scanning trending server: %v", err)
			continue
		}

		// Parse the JSON value
		var value map[string]interface{}
		if err := json.Unmarshal(valueJSON, &value); err != nil {
			log.Printf("Error parsing server JSON: %v", err)
			continue
		}

		// Add enhanced fields
		value["id"] = serverName
		value["name"] = serverName
		value["stats"] = map[string]interface{}{
			"rating":         rating,
			"rating_count":   ratingCount,
			"install_count":  installCount,
			"trending_score": trendingScore,
		}

		trending = append(trending, value)
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")

	// Write response
	response := map[string]interface{}{
		"trending":  trending,
		"period":    "30_days",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleServerDetail handles /v0/servers/{id} to get individual server details
func (h *EnhancedHandler) HandleServerDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID from path: /v0/servers/{id}
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	serverID := path
	if serverID == "" {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// Query the registry database for this specific server
	query := `
		SELECT
			s.server_name,
			s.value,
			COALESCE(ss.rating, 0) as rating,
			COALESCE(ss.rating_count, 0) as rating_count,
			COALESCE(ss.installation_count, 0) as installation_count
		FROM servers s
		LEFT JOIN proxy_server_stats ss ON s.server_name = ss.server_id
		WHERE s.server_name = $1 AND s.is_latest = true
		LIMIT 1
	`

	ctx := r.Context()
	rows, err := h.registryDB.QueryContext(ctx, query, serverID)
	if err != nil {
		log.Printf("Database query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	var (
		serverName   string
		valueJSON    []byte
		rating       float64
		ratingCount  int
		installCount int
	)

	if err := rows.Scan(&serverName, &valueJSON, &rating, &ratingCount, &installCount); err != nil {
		log.Printf("Error scanning row: %v", err)
		http.Error(w, "Error reading server data", http.StatusInternalServerError)
		return
	}

	// Parse the JSON value
	var value map[string]interface{}
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		log.Printf("Error parsing server JSON: %v", err)
		http.Error(w, "Error parsing server data", http.StatusInternalServerError)
		return
	}

	// Add enhanced fields
	value["id"] = serverName
	value["name"] = serverName
	value["stats"] = map[string]interface{}{
		"rating":        rating,
		"rating_count":  ratingCount,
		"install_count": installCount,
	}

	// Set headers and write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// Helper functions
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}