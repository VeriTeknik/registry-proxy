package models

import (
	"time"
)

// ServerStatus represents the lifecycle status of a server
type ServerStatus string

const (
	ServerStatusActive     ServerStatus = "active"
	ServerStatusDeprecated ServerStatus = "deprecated"
)

// Repository represents a source code repository
type Repository struct {
	URL    string `json:"url" bson:"url"`
	Source string `json:"source" bson:"source"`
	ID     string `json:"id" bson:"id"`
}

// VersionDetail represents version information
type VersionDetail struct {
	Version     string `json:"version" bson:"version"`
	ReleaseDate string `json:"release_date" bson:"release_date"`
	IsLatest    bool   `json:"is_latest" bson:"is_latest"`
}

// Package represents package information
type Package struct {
	RegistryName         string                 `json:"registry_name" bson:"registry_name"`
	Name                 string                 `json:"name" bson:"name"`
	Version              string                 `json:"version" bson:"version"`
	RuntimeHint          string                 `json:"runtime_hint,omitempty" bson:"runtime_hint,omitempty"`
	EnvironmentVariables []EnvironmentVariable  `json:"environment_variables,omitempty" bson:"environment_variables,omitempty"`
	PackageArguments     []map[string]any       `json:"package_arguments,omitempty" bson:"package_arguments,omitempty"`
	RuntimeArguments     []map[string]any       `json:"runtime_arguments,omitempty" bson:"runtime_arguments,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name        string `json:"name" bson:"name"`
	Description string `json:"description,omitempty" bson:"description,omitempty"`
	IsRequired  bool   `json:"is_required,omitempty" bson:"is_required,omitempty"`
	IsSecret    bool   `json:"is_secret,omitempty" bson:"is_secret,omitempty"`
}

// Server represents basic server information
type Server struct {
	ID            string        `json:"id" bson:"id"`
	Name          string        `json:"name" bson:"name"`
	Description   string        `json:"description" bson:"description"`
	Status        ServerStatus  `json:"status,omitempty" bson:"status,omitempty"`
	Repository    Repository    `json:"repository" bson:"repository"`
	VersionDetail VersionDetail `json:"version_detail" bson:"version_detail"`
}

// ServerDetail represents detailed server information
type ServerDetail struct {
	Server   `json:",inline" bson:",inline"`
	Packages []Package `json:"packages,omitempty" bson:"packages,omitempty"`
	Remotes  []Remote  `json:"remotes,omitempty" bson:"remotes,omitempty"`
}

// Remote represents remote server configuration
type Remote struct {
	TransportType string `json:"transport_type" bson:"transport_type"`
	URL           string `json:"url" bson:"url"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string    `json:"id" bson:"_id"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	User      string    `json:"user" bson:"user"`
	Action    string    `json:"action" bson:"action"`
	ServerID  string    `json:"server_id,omitempty" bson:"server_id,omitempty"`
	Details   string    `json:"details" bson:"details"`
	IP        string    `json:"ip" bson:"ip"`
}

// User represents an admin user
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}