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

// Transport represents the transport mechanism for a package
type Transport struct {
	Type string `json:"type"` // stdio, sse, or http
}

// Argument represents a runtime or package argument
type Argument struct {
	Type        string   `json:"type"`                  // "positional" or "named"
	Name        string   `json:"name,omitempty"`        // For named arguments
	Value       string   `json:"value,omitempty"`       // The argument value
	Default     string   `json:"default,omitempty"`     // Default value
	Description string   `json:"description,omitempty"` // Description of the argument
	Choices     []string `json:"choices,omitempty"`     // Valid choices for the argument
	IsRequired  bool     `json:"is_required,omitempty"` // Whether the argument is required
}

// Package represents package information
type Package struct {
	RegistryName         string                 `json:"registry_name"`
	Name                 string                 `json:"name"`
	Version              string                 `json:"version"`
	Transport            *Transport             `json:"transport,omitempty"`
	RuntimeHint          string                 `json:"runtime_hint,omitempty"`
	RuntimeArguments     []Argument             `json:"runtime_arguments,omitempty"`
	PackageArguments     []Argument             `json:"package_arguments,omitempty"`
	EnvironmentVariables []EnvironmentVariable  `json:"environment_variables,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	Default     string `json:"default,omitempty"`
	IsRequired  bool   `json:"is_required,omitempty"`
	IsSecret    bool   `json:"is_secret,omitempty"`
}

// RemoteHeader represents a header for remote servers
type RemoteHeader struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	IsRequired  bool   `json:"is_required,omitempty"`
	IsSecret    bool   `json:"is_secret,omitempty"`
}

// Remote represents a remote server configuration
type Remote struct {
	TransportType string         `json:"transport_type"` // "sse", "streamable-http", "http"
	URL           string         `json:"url"`
	Headers       []RemoteHeader `json:"headers,omitempty"`
}

// ServerDetail represents detailed server information
type ServerDetail struct {
	Server
	Packages []Package `json:"packages,omitempty"`
	Remotes  []Remote  `json:"remotes,omitempty"`
}

// EnrichedServer combines Server with Packages and Stats for the proxy response
type EnrichedServer struct {
	Server
	Packages          []Package `json:"packages,omitempty"`
	Remotes           []Remote  `json:"remotes,omitempty"`
	Rating            float64   `json:"rating,omitempty"`
	RatingCount       int       `json:"rating_count,omitempty"`
	InstallationCount int       `json:"installation_count,omitempty"`
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