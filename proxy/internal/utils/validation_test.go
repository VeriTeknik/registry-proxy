package utils

import (
	"strings"
	"testing"
)

func TestValidateServerID(t *testing.T) {
	tests := []struct {
		name     string
		serverID string
		wantErr  bool
	}{
		{
			name:     "valid UUID",
			serverID: "550e8400-e29b-41d4-a716-446655440000",
			wantErr:  false,
		},
		{
			name:     "valid qualified name",
			serverID: "io.github.user.repo",
			wantErr:  false,
		},
		{
			name:     "valid name with hyphens",
			serverID: "my-server-name",
			wantErr:  false,
		},
		{
			name:     "valid name with underscores",
			serverID: "my_server_name",
			wantErr:  false,
		},
		{
			name:     "valid alphanumeric",
			serverID: "server123",
			wantErr:  false,
		},
		{
			name:     "empty string",
			serverID: "",
			wantErr:  true,
		},
		{
			name:     "path traversal attempt",
			serverID: "../../../etc/passwd",
			wantErr:  true,
		},
		{
			name:     "path traversal with dots",
			serverID: "..\\..\\windows\\system32",
			wantErr:  true,
		},
		{
			name:     "forward slash",
			serverID: "user/repo",
			wantErr:  true,
		},
		{
			name:     "backslash",
			serverID: "user\\repo",
			wantErr:  true,
		},
		{
			name:     "too long (256 chars)",
			serverID: strings.Repeat("a", 256),
			wantErr:  true,
		},
		{
			name:     "exactly 255 chars (max allowed)",
			serverID: strings.Repeat("a", 255),
			wantErr:  false,
		},
		{
			name:     "starts with dot",
			serverID: ".hidden",
			wantErr:  true,
		},
		{
			name:     "special characters",
			serverID: "server@#$%",
			wantErr:  true,
		},
		{
			name:     "sql injection attempt",
			serverID: "'; DROP TABLE servers--",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerID(tt.serverID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSortParameter(t *testing.T) {
	validSorts := []string{"created", "updated", "rating_desc", "installs_desc", "trending"}

	tests := []struct {
		name    string
		sort    string
		wantErr bool
	}{
		{
			name:    "empty sort (default)",
			sort:    "",
			wantErr: false,
		},
		{
			name:    "valid sort: created",
			sort:    "created",
			wantErr: false,
		},
		{
			name:    "valid sort: rating_desc",
			sort:    "rating_desc",
			wantErr: false,
		},
		{
			name:    "invalid sort",
			sort:    "invalid_option",
			wantErr: true,
		},
		{
			name:    "sql injection in sort",
			sort:    "name; DROP TABLE servers",
			wantErr: true,
		},
		{
			name:    "path traversal in sort",
			sort:    "../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSortParameter(tt.sort, validSorts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSortParameter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStruct_PaginationRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     PaginationRequest
		wantErr bool
	}{
		{
			name: "valid pagination",
			req: PaginationRequest{
				Limit:  20,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "max limit",
			req: PaginationRequest{
				Limit:  1000,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "limit too high",
			req: PaginationRequest{
				Limit:  1001,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "limit zero",
			req: PaginationRequest{
				Limit:  0,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "negative offset",
			req: PaginationRequest{
				Limit:  20,
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "large offset",
			req: PaginationRequest{
				Limit:  20,
				Offset: 999999,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct(PaginationRequest) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStruct_ServerFilterRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     ServerFilterRequest
		wantErr bool
	}{
		{
			name: "valid filter",
			req: ServerFilterRequest{
				Search:        "test server",
				Category:      "data-analysis",
				MinRating:     3.5,
				MinInstalls:   10,
				RegistryTypes: []string{"npm", "pypi"},
				Tags:          []string{"ai", "automation"},
				HasTransport:  []string{"stdio"},
			},
			wantErr: false,
		},
		{
			name: "search too long",
			req: ServerFilterRequest{
				Search: strings.Repeat("a", 201),
			},
			wantErr: true,
		},
		{
			name: "category too long",
			req: ServerFilterRequest{
				Category: strings.Repeat("a", 101),
			},
			wantErr: true,
		},
		{
			name: "rating too low",
			req: ServerFilterRequest{
				MinRating: -1,
			},
			wantErr: true,
		},
		{
			name: "rating too high",
			req: ServerFilterRequest{
				MinRating: 6,
			},
			wantErr: true,
		},
		{
			name: "invalid registry type",
			req: ServerFilterRequest{
				RegistryTypes: []string{"invalid_registry"},
			},
			wantErr: true,
		},
		{
			name: "valid registry types",
			req: ServerFilterRequest{
				RegistryTypes: []string{"npm", "pypi", "oci", "mcpb", "nuget", "remote"},
			},
			wantErr: false,
		},
		{
			name: "invalid transport type",
			req: ServerFilterRequest{
				HasTransport: []string{"invalid_transport"},
			},
			wantErr: true,
		},
		{
			name: "valid transport types",
			req: ServerFilterRequest{
				HasTransport: []string{"stdio", "sse", "http"},
			},
			wantErr: false,
		},
		{
			name: "tag too long",
			req: ServerFilterRequest{
				Tags: []string{strings.Repeat("a", 51)},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct(ServerFilterRequest) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStruct_RatingRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     RatingRequest
		wantErr bool
	}{
		{
			name: "valid rating with comment",
			req: RatingRequest{
				Rating:  4,
				Comment: "Great server!",
			},
			wantErr: false,
		},
		{
			name: "valid rating without comment",
			req: RatingRequest{
				Rating: 5,
			},
			wantErr: false,
		},
		{
			name: "rating too low",
			req: RatingRequest{
				Rating: 0,
			},
			wantErr: true,
		},
		{
			name: "rating too high",
			req: RatingRequest{
				Rating: 6,
			},
			wantErr: true,
		},
		{
			name: "comment too long",
			req: RatingRequest{
				Rating:  4,
				Comment: strings.Repeat("a", 1001),
			},
			wantErr: true,
		},
		{
			name: "missing rating",
			req: RatingRequest{
				Comment: "Comment without rating",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct(RatingRequest) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStruct_InstallRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     InstallRequest
		wantErr bool
	}{
		{
			name: "valid install request",
			req: InstallRequest{
				Version:  "1.0.0",
				Platform: "linux",
			},
			wantErr: false,
		},
		{
			name:    "empty install request",
			req:     InstallRequest{},
			wantErr: false, // Both fields are optional
		},
		{
			name: "version too long",
			req: InstallRequest{
				Version: strings.Repeat("1.0.0-", 20),
			},
			wantErr: true,
		},
		{
			name: "platform too long",
			req: InstallRequest{
				Platform: strings.Repeat("linux", 20),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct(InstallRequest) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateServerID(b *testing.B) {
	serverID := "io.github.user.repo"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateServerID(serverID)
	}
}

func BenchmarkValidateStruct(b *testing.B) {
	req := ServerFilterRequest{
		Search:        "test",
		Category:      "data",
		MinRating:     3.0,
		RegistryTypes: []string{"npm", "pypi"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateStruct(&req)
	}
}
