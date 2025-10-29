package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/veriteknik/registry-proxy/internal/db"
)

// RatingsHandler handles rating and installation tracking
type RatingsHandler struct {
	db    *db.DB
	cache Cache
}

// Cache interface for invalidating cached server data
type Cache interface {
	Clear()
}

// NewRatingsHandler creates a new ratings handler
func NewRatingsHandler(database *db.DB, cache Cache) *RatingsHandler {
	return &RatingsHandler{
		db:    database,
		cache: cache,
	}
}

// RatingRequest represents a rating submission
type RatingRequest struct {
	Rating    int    `json:"rating"`
	UserID    string `json:"user_id"`
	Comment   string `json:"comment"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

// InstallRequest represents an installation tracking request
type InstallRequest struct {
	UserID   string `json:"user_id"`
	Source   string `json:"source"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

// StatsResponse represents server statistics
type StatsResponse struct {
	Stats struct {
		ServerID          string  `json:"server_id"`
		InstallationCount int     `json:"installation_count"`
		Rating            float64 `json:"rating"`
		RatingCount       int     `json:"rating_count"`
	} `json:"stats"`
}

// HandleRate handles POST /v0/servers/:id/rate
func (h *RatingsHandler) HandleRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID from path: /v0/servers/{id}/rate
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	serverID := strings.TrimSuffix(path, "/rate")
	if serverID == "" || serverID == path {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req RatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate rating
	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Validate user ID
	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Save rating to database
	if err := h.db.UpsertRating(r.Context(), serverID, req.UserID, req.Rating, req.Comment); err != nil {
		log.Printf("Failed to save rating: %v", err)
		http.Error(w, "Failed to save rating", http.StatusInternalServerError)
		return
	}

	// Invalidate cache so updated stats are immediately visible to all users
	if h.cache != nil {
		h.cache.Clear()
		log.Printf("Cache cleared after rating submission for server: %s", serverID)
	}

	// Get updated stats
	rating, ratingCount, installCount, err := h.db.GetServerStats(r.Context(), serverID)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		// Don't fail the request, just return success without stats
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Rating saved successfully",
		}); err != nil {
			log.Printf("Error encoding rating response: %v", err)
		}
		return
	}

	// Return success with updated stats
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"stats": map[string]interface{}{
			"server_id":          serverID,
			"rating":             rating,
			"rating_count":       ratingCount,
			"installation_count": installCount,
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding rating response: %v", err)
	}
}

// HandleInstall handles POST /v0/servers/:id/install
func (h *RatingsHandler) HandleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID from path: /v0/servers/{id}/install
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	serverID := strings.TrimSuffix(path, "/install")
	if serverID == "" || serverID == path {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Use anonymous if no user ID provided
	userID := req.UserID
	if userID == "" {
		userID = "anonymous"
	}

	// Track installation
	if err := h.db.TrackInstallation(r.Context(), serverID, userID, req.Source, req.Version, req.Platform); err != nil {
		log.Printf("Failed to track installation: %v", err)
		http.Error(w, "Failed to track installation", http.StatusInternalServerError)
		return
	}

	// Get updated stats
	rating, ratingCount, installCount, err := h.db.GetServerStats(r.Context(), serverID)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		// Don't fail the request, just return success without stats
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Installation tracked successfully",
		}); err != nil {
			log.Printf("Error encoding install response: %v", err)
		}
		return
	}

	// Return success with updated stats
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"stats": map[string]interface{}{
			"server_id":          serverID,
			"rating":             rating,
			"rating_count":       ratingCount,
			"installation_count": installCount,
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding install response: %v", err)
	}
}

// HandleStats handles GET /v0/servers/:id/stats
func (h *RatingsHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID from path: /v0/servers/{id}/stats
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	serverID := strings.TrimSuffix(path, "/stats")
	if serverID == "" || serverID == path {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Get stats from database
	rating, ratingCount, installCount, err := h.db.GetServerStats(r.Context(), serverID)
	if err != nil {
		log.Printf("Failed to get stats for %s: %v", serverID, err)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	// Return stats
	var response StatsResponse
	response.Stats.ServerID = serverID
	response.Stats.Rating = rating
	response.Stats.RatingCount = ratingCount
	response.Stats.InstallationCount = installCount

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding stats response: %v", err)
	}
}

// HandleGetReviews handles GET /v0/servers/:id/reviews
func (h *RatingsHandler) HandleGetReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	// Extract server ID from path: /v0/servers/{id}/reviews
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	serverID := strings.TrimSuffix(path, "/reviews")
	if serverID == "" || serverID == path {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid path",
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	// Get reviews from database
	reviews, err := h.db.GetReviews(r.Context(), serverID)
	if err != nil {
		log.Printf("Failed to get reviews for %s: %v", serverID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to get reviews",
			"reviews": []interface{}{},
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	// Ensure reviews is never nil (return empty array instead)
	if reviews == nil {
		reviews = []db.Review{}
	}

	// Return reviews
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"reviews": reviews,
	}); err != nil {
		log.Printf("Error encoding reviews response: %v", err)
	}
}

// FeedbackItem represents a feedback item in the format expected by the frontend
type FeedbackItem struct {
	ID           string `json:"id"`
	ServerID     string `json:"server_id"`
	Source       string `json:"source"`
	UserID       string `json:"user_id"`
	Username     string `json:"username,omitempty"`
	UserAvatar   string `json:"user_avatar,omitempty"`
	Rating       int    `json:"rating"`
	Comment      string `json:"comment,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// HandleGetFeedback handles GET /v0/servers/:id/feedback with pagination
func (h *RatingsHandler) HandleGetFeedback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Method not allowed",
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	// Extract server ID from path: /v0/servers/{id}/feedback
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	serverID := strings.TrimSuffix(path, "/feedback")
	if serverID == "" || serverID == path {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid path",
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	limit := 20
	if l := query.Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	sort := query.Get("sort")
	if sort == "" {
		sort = "newest"
	}

	// Get paginated reviews from database
	reviews, totalCount, err := h.db.GetReviewsPaginated(r.Context(), serverID, limit, offset, sort)
	if err != nil {
		log.Printf("Failed to get feedback for %s: %v", serverID, err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "Failed to get feedback",
			"feedback":    []interface{}{},
			"total_count": 0,
			"has_more":    false,
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	// Ensure reviews is never nil (return empty array instead)
	if reviews == nil {
		reviews = []db.Review{}
	}

	// Transform reviews to feedback items
	feedbackItems := make([]FeedbackItem, len(reviews))
	for i, review := range reviews {
		feedbackItems[i] = FeedbackItem{
			ID:         review.UUID,
			ServerID:   review.ServerExternalID,
			Source:     review.ServerSource,
			UserID:     review.UserID,
			Username:   "", // TODO: Fetch from user service if needed
			UserAvatar: "", // TODO: Fetch from user service if needed
			Rating:     review.Rating,
			Comment:    review.Comment,
			CreatedAt:  review.CreatedAt.Format("2006-01-02T15:04:05.999999Z07:00"),
			UpdatedAt:  review.UpdatedAt.Format("2006-01-02T15:04:05.999999Z07:00"),
		}
	}

	// Calculate has_more
	hasMore := offset+len(reviews) < totalCount

	// Return feedback response
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"feedback":    feedbackItems,
		"total_count": totalCount,
		"has_more":    hasMore,
	}); err != nil {
		log.Printf("Error encoding feedback response: %v", err)
	}
}

// HandleGetUserRating handles GET /v0/servers/:id/rating/:userId
func (h *RatingsHandler) HandleGetUserRating(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID and user ID from path: /v0/servers/{id}/rating/{userId}
	path := strings.TrimPrefix(r.URL.Path, "/v0/servers/")
	parts := strings.Split(path, "/")

	if len(parts) < 3 || parts[len(parts)-2] != "rating" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Server ID could contain slashes (e.g., io.github.user/repo)
	// So we join all parts except the last two (rating/userId)
	serverParts := parts[:len(parts)-2]
	serverID := strings.Join(serverParts, "/")
	userID := parts[len(parts)-1]

	// Get user's rating from database
	ctx := r.Context()
	var rating int
	var comment string
	var createdAt time.Time

	query := `
		SELECT rating, comment, created_at
		FROM proxy_user_ratings
		WHERE server_id = $1 AND user_id = $2
		LIMIT 1
	`

	err := h.db.QueryRowContext(ctx, query, serverID, userID).Scan(&rating, &comment, &createdAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// User hasn't rated yet
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"has_rated": false,
			}); err != nil {
				log.Printf("Error encoding response: %v", err)
			}
			return
		}

		log.Printf("Database error checking user rating: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return user's rating
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"has_rated": true,
		"feedback": map[string]interface{}{
			"id":         fmt.Sprintf("%s:%s", serverID, userID),
			"rating":     rating,
			"comment":    comment,
			"created_at": createdAt.Format(time.RFC3339),
		},
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
