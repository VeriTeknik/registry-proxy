package models

import "time"

// Repository represents a source code repository
type Repository struct {
	URL    string `json:"url"`
	Source string `json:"source"`
	ID     string `json:"id"`
}

// VersionDetail represents the version details of a server
type VersionDetail struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"release_date"`
	IsLatest    bool   `json:"is_latest"`
}

// Server represents basic server information from the registry
type Server struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Status        string        `json:"status,omitempty"`
	Repository    Repository    `json:"repository"`
	VersionDetail VersionDetail `json:"version_detail"`
}

// Package represents package information
type Package struct {
	RegistryName         string                 `json:"registry_name"`
	Name                 string                 `json:"name"`
	Version              string                 `json:"version"`
	EnvironmentVariables []EnvironmentVariable  `json:"environment_variables,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
}

// ServerDetail represents detailed server information
type ServerDetail struct {
	Server
	Packages []Package `json:"packages,omitempty"`
}

// EnrichedServer combines Server with Packages for the proxy response
type EnrichedServer struct {
	Server
	Packages []Package `json:"packages,omitempty"`
}

// ProxyResponse wraps the enriched servers with metadata
type ProxyResponse struct {
	Servers  []EnrichedServer `json:"servers"`
	Metadata ResponseMetadata `json:"metadata"`
}

// ResponseMetadata contains pagination and filtering info
type ResponseMetadata struct {
	NextCursor   string    `json:"next_cursor,omitempty"`
	Count        int       `json:"count"`
	Total        int       `json:"total,omitempty"`
	FilteredBy   string    `json:"filtered_by,omitempty"`
	SortedBy     string    `json:"sorted_by,omitempty"`
	CachedAt     time.Time `json:"cached_at,omitempty"`
}