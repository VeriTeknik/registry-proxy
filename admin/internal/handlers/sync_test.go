package handlers

import (
	"testing"

	"github.com/pluggedin/registry-admin/internal/models"
)

func TestFilterLatestVersions(t *testing.T) {
	items := []OfficialServerItem{
		{
			Server: OfficialServer{
				Name:        "server-a",
				Description: "Latest version of A",
				Version:     "2.0.0",
			},
			Meta: OfficialMeta{
				Official: OfficialMetadata{IsLatest: true},
			},
		},
		{
			Server: OfficialServer{
				Name:        "server-a",
				Description: "Old version of A",
				Version:     "1.0.0",
			},
			Meta: OfficialMeta{
				Official: OfficialMetadata{IsLatest: false},
			},
		},
		{
			Server: OfficialServer{
				Name:        "server-b",
				Description: "Only version of B",
				Version:     "1.0.0",
			},
			Meta: OfficialMeta{
				Official: OfficialMetadata{IsLatest: true},
			},
		},
	}

	result := filterLatestVersions(items)

	if len(result) != 2 {
		t.Fatalf("Expected 2 latest servers, got %d", len(result))
	}
	if result[0].Name != "server-a" {
		t.Errorf("Expected first server to be 'server-a', got '%s'", result[0].Name)
	}
	if result[0].VersionDetail.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", result[0].VersionDetail.Version)
	}
	if result[1].Name != "server-b" {
		t.Errorf("Expected second server to be 'server-b', got '%s'", result[1].Name)
	}
}

func TestFilterLatestVersions_Empty(t *testing.T) {
	result := filterLatestVersions([]OfficialServerItem{})
	if len(result) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(result))
	}
}

func TestFilterLatestVersions_NoneLatest(t *testing.T) {
	items := []OfficialServerItem{
		{
			Server: OfficialServer{Name: "server-a", Version: "1.0.0"},
			Meta:   OfficialMeta{Official: OfficialMetadata{IsLatest: false}},
		},
	}
	result := filterLatestVersions(items)
	if len(result) != 0 {
		t.Errorf("Expected 0 results when none are latest, got %d", len(result))
	}
}

func TestConvertOfficialToServerDetail(t *testing.T) {
	official := &OfficialServer{
		Name:        "test/mcp-server",
		Description: "A test MCP server",
		Version:     "2.1.0",
		Repository: models.Repository{
			URL:    "https://github.com/test/repo",
			Source: "github",
			ID:     "test/repo",
		},
		Packages: []OfficialPackage{
			{
				RegistryType: "npm",
				Identifier:   "@test/mcp-server",
				Version:      "2.1.0",
				RuntimeHint:  "node",
				Transport:    map[string]interface{}{"type": "stdio"},
				EnvironmentVariables: []models.EnvironmentVariable{
					{Name: "API_KEY", Description: "API key", IsRequired: true, IsSecret: true},
				},
			},
		},
		Remotes: []OfficialRemote{
			{Type: "streamable-http", URL: "https://example.com/mcp"},
		},
	}

	result := convertOfficialToServerDetail(official)

	if result.Name != "test/mcp-server" {
		t.Errorf("Name = %q, want %q", result.Name, "test/mcp-server")
	}
	if result.Description != "A test MCP server" {
		t.Errorf("Description = %q, want %q", result.Description, "A test MCP server")
	}
	if result.VersionDetail.Version != "2.1.0" {
		t.Errorf("Version = %q, want %q", result.VersionDetail.Version, "2.1.0")
	}
	if !result.VersionDetail.IsLatest {
		t.Error("Expected IsLatest to be true")
	}
	if result.Repository.URL != "https://github.com/test/repo" {
		t.Errorf("Repository.URL = %q, want %q", result.Repository.URL, "https://github.com/test/repo")
	}

	// Verify packages
	if len(result.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(result.Packages))
	}
	pkg := result.Packages[0]
	if pkg.RegistryName != "npm" {
		t.Errorf("Package.RegistryName = %q, want %q", pkg.RegistryName, "npm")
	}
	if pkg.Name != "@test/mcp-server" {
		t.Errorf("Package.Name = %q, want %q", pkg.Name, "@test/mcp-server")
	}
	if pkg.Transport == nil || pkg.Transport.Type != "stdio" {
		t.Error("Expected transport type 'stdio'")
	}
	if len(pkg.EnvironmentVariables) != 1 {
		t.Fatalf("Expected 1 env var, got %d", len(pkg.EnvironmentVariables))
	}

	// Verify remotes
	if len(result.Remotes) != 1 {
		t.Fatalf("Expected 1 remote, got %d", len(result.Remotes))
	}
	if result.Remotes[0].TransportType != "streamable-http" {
		t.Errorf("Remote.TransportType = %q, want %q", result.Remotes[0].TransportType, "streamable-http")
	}
}

func TestConvertOfficialToServerDetail_NoTransport(t *testing.T) {
	official := &OfficialServer{
		Name:    "simple-server",
		Version: "1.0.0",
		Packages: []OfficialPackage{
			{
				RegistryType: "pypi",
				Identifier:   "simple-server",
				// No Transport field
			},
		},
	}

	result := convertOfficialToServerDetail(official)

	if len(result.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(result.Packages))
	}
	if result.Packages[0].Transport != nil {
		t.Error("Expected nil transport when not provided")
	}
}

func TestExtractServerTypes(t *testing.T) {
	tests := []struct {
		name     string
		server   *models.ServerDetail
		expected []string
	}{
		{
			name: "single npm package",
			server: &models.ServerDetail{
				Server: models.Server{Name: "test"},
				Packages: []models.Package{
					{RegistryName: "npm"},
				},
			},
			expected: []string{"npm"},
		},
		{
			name: "oci maps to docker",
			server: &models.ServerDetail{
				Server: models.Server{Name: "test"},
				Packages: []models.Package{
					{RegistryName: "oci"},
				},
			},
			expected: []string{"docker"},
		},
		{
			name: "remote type fallback when no packages",
			server: &models.ServerDetail{
				Server: models.Server{Name: "test"},
				Remotes: []models.Remote{
					{TransportType: "streamable-http"},
				},
			},
			expected: []string{"remote"},
		},
		{
			name: "deduplicates types",
			server: &models.ServerDetail{
				Server: models.Server{Name: "test"},
				Packages: []models.Package{
					{RegistryName: "npm"},
					{RegistryName: "npm"},
				},
			},
			expected: []string{"npm"},
		},
		{
			name: "no packages or remotes",
			server: &models.ServerDetail{
				Server: models.Server{Name: "test"},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServerTypes(tt.server)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d types, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Type[%d] = %q, want %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestShouldUpdate(t *testing.T) {
	h := &SyncHandler{}

	tests := []struct {
		name     string
		existing *models.ServerDetail
		official *models.ServerDetail
		expected bool
	}{
		{
			name: "different versions",
			existing: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
			},
			official: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "2.0.0"},
				},
			},
			expected: true,
		},
		{
			name: "same version same packages",
			existing: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
				Packages: []models.Package{{Name: "pkg", Version: "1.0.0"}},
			},
			official: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
				Packages: []models.Package{{Name: "pkg", Version: "1.0.0"}},
			},
			expected: false,
		},
		{
			name: "different package count",
			existing: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
				Packages: []models.Package{{Name: "pkg1"}},
			},
			official: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
				Packages: []models.Package{{Name: "pkg1"}, {Name: "pkg2"}},
			},
			expected: true,
		},
		{
			name: "new remotes added",
			existing: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
			},
			official: &models.ServerDetail{
				Server: models.Server{
					VersionDetail: models.VersionDetail{Version: "1.0.0"},
				},
				Remotes: []models.Remote{{TransportType: "sse", URL: "https://example.com"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.shouldUpdate(tt.existing, tt.official)
			if result != tt.expected {
				t.Errorf("shouldUpdate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildPaginatedURL(t *testing.T) {
	h := &SyncHandler{officialRegistryURL: "https://registry.example.com"}

	// Without cursor
	url := h.buildPaginatedURL("", 100)
	expected := "https://registry.example.com/v0/servers?limit=100"
	if url != expected {
		t.Errorf("buildPaginatedURL(\"\", 100) = %q, want %q", url, expected)
	}

	// With cursor
	url = h.buildPaginatedURL("abc123", 50)
	expected = "https://registry.example.com/v0/servers?limit=50&cursor=abc123"
	if url != expected {
		t.Errorf("buildPaginatedURL(\"abc123\", 50) = %q, want %q", url, expected)
	}
}

func TestMapRegistryTypeToFriendlyName(t *testing.T) {
	tests := map[string]string{
		"oci":             "docker",
		"pypi":            "python",
		"npm":             "npm",
		"sse":             "remote",
		"streamable-http": "remote",
		"unknown":         "unknown",
		"":                "",
	}

	for input, expected := range tests {
		result := mapRegistryTypeToFriendlyName(input)
		if result != expected {
			t.Errorf("mapRegistryTypeToFriendlyName(%q) = %q, want %q", input, result, expected)
		}
	}
}
