package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/veriteknik/registry-proxy/internal/db"
	"github.com/veriteknik/registry-proxy/internal/utils"
	"go.uber.org/zap"
)

// EnhancedHandler handles enhanced endpoints that query the registry database directly
type EnhancedHandler struct {
	registryDB *db.DB
	proxyDB    *db.DB
	logger     *zap.Logger
}

// NewEnhancedHandler creates a new enhanced handler
func NewEnhancedHandler(registryDB, proxyDB *db.DB) *EnhancedHandler {
	return &EnhancedHandler{
		registryDB: registryDB,
		proxyDB:    proxyDB,
		logger:     utils.Logger,
	}
}

// HandleEnhancedServers handles /v0/enhanced/servers with filtering and sorting
func (h *EnhancedHandler) HandleEnhancedServers(w http.ResponseWriter, r *http.Request) {
	if !utils.RequireMethod(w, r, http.MethodGet) {
		return
	}

	// Parse query parameters using utility functions
	query := r.URL.Query()

	// Create filter from query parameters
	filter := db.ServerFilter{
		Search:        query.Get("search"),
		Category:      query.Get("category"),
		MinRating:     utils.ParseFloatParam(query, "min_rating"),
		MinInstalls:   utils.ParseIntParam(query, "min_installs", 0, 0),
		RegistryTypes: utils.ParseList(query, "registry_types"),
		Tags:          utils.ParseList(query, "tags"),
		HasTransport:  utils.ParseList(query, "transports"),
	}

	// Validate filter
	filterReq := utils.ServerFilterRequest{
		Search:        filter.Search,
		Category:      filter.Category,
		MinRating:     filter.MinRating,
		MinInstalls:   filter.MinInstalls,
		RegistryTypes: filter.RegistryTypes,
		Tags:          filter.Tags,
		HasTransport:  filter.HasTransport,
	}
	if err := utils.ValidateStruct(&filterReq); err != nil {
		h.logger.Warn("Invalid filter parameters", zap.Error(err))
		utils.WriteJSONError(w, "Invalid filter parameters", http.StatusBadRequest)
		return
	}

	// Parse and validate pagination
	limit := utils.ParseIntParam(query, "limit", 20, 1000)
	offset := utils.ParseIntParam(query, "offset", 0, 0)

	paginationReq := utils.PaginationRequest{
		Limit:  limit,
		Offset: offset,
	}
	if err := utils.ValidateStruct(&paginationReq); err != nil {
		h.logger.Warn("Invalid pagination parameters", zap.Error(err))
		utils.WriteJSONError(w, "Invalid pagination parameters", http.StatusBadRequest)
		return
	}

	// Get and validate sort parameter
	sort := query.Get("sort")
	if sort == "" {
		sort = "created"
	}

	// Query enhanced servers
	servers, totalCount, err := h.registryDB.QueryServersEnhanced(r.Context(), filter, sort, limit, offset)
	if err != nil {
		h.logger.Error("Failed to query enhanced servers", zap.Error(err))
		utils.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
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
	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))

	// Write JSON response
	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error("Failed to write response", zap.Error(err))
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
	if !utils.RequireMethod(w, r, http.MethodGet) {
		return
	}

	// Extract and validate server ID from path: /v0/servers/{id}
	path := r.URL.Path[len("/v0/servers/"):]
	serverID := path

	// Validate server ID to prevent path traversal and injection attacks
	if err := utils.ValidateServerID(serverID); err != nil {
		h.logger.Warn("Invalid server ID", zap.String("server_id", serverID), zap.Error(err))
		utils.WriteJSONError(w, "Invalid server ID", http.StatusBadRequest)
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
		h.logger.Error("Database query failed", zap.Error(err))
		utils.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		utils.WriteJSONError(w, "Server not found", http.StatusNotFound)
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
		h.logger.Error("Failed to scan row", zap.Error(err))
		utils.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Parse and enrich server data using helper
	stats := db.ServerStats{
		Rating:            rating,
		RatingCount:       ratingCount,
		InstallationCount: installCount,
	}

	var value map[string]interface{}
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		h.logger.Error("Failed to parse server JSON", zap.Error(err))
		utils.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
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

	// Write JSON response
	if err := utils.WriteJSON(w, http.StatusOK, value); err != nil {
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