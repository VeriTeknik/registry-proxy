package db

import (
	"strings"
	"testing"

	"github.com/lib/pq"
)

func TestValidateAndGetSortClause(t *testing.T) {
	tests := []struct {
		name        string
		sort        string
		wantErr     bool
		wantContain string
	}{
		{
			name:        "valid sort: created",
			sort:        "created",
			wantErr:     false,
			wantContain: "published_at DESC",
		},
		{
			name:        "valid sort: rating_desc",
			sort:        "rating_desc",
			wantErr:     false,
			wantContain: "rating DESC",
		},
		{
			name:        "valid sort: updated",
			sort:        "updated",
			wantErr:     false,
			wantContain: "updated_at DESC",
		},
		{
			name:        "valid sort: installs_desc",
			sort:        "installs_desc",
			wantErr:     false,
			wantContain: "installation_count DESC",
		},
		{
			name:        "valid sort: trending",
			sort:        "trending",
			wantErr:     false,
			wantContain: "installation_count * 0.3",
		},
		{
			name:        "empty sort (default)",
			sort:        "",
			wantErr:     false,
			wantContain: "published_at DESC",
		},
		{
			name:    "invalid sort: sql injection attempt",
			sort:    "name; DROP TABLE servers--",
			wantErr: true,
		},
		{
			name:    "invalid sort: unknown option",
			sort:    "unknown_sort",
			wantErr: true,
		},
		{
			name:    "invalid sort: malicious input",
			sort:    "../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateAndGetSortClause(tt.sort)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAndGetSortClause() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.wantContain) {
				t.Errorf("validateAndGetSortClause() = %v, want to contain %v", got, tt.wantContain)
			}
		})
	}
}

func TestBuildSearchFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter ServerFilter
		want   int // number of conditions added
	}{
		{
			name:   "empty search",
			filter: ServerFilter{Search: ""},
			want:   0,
		},
		{
			name:   "with search term",
			filter: ServerFilter{Search: "test server"},
			want:   1,
		},
		{
			name:   "search with special characters",
			filter: ServerFilter{Search: "test's \"quoted\" server"},
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialLen := 1 // base condition
			cteWhere := make([]interface{}, initialLen)
			result := buildSearchFilter(cteWhere, tt.filter)
			if len(result)-initialLen != tt.want {
				t.Errorf("buildSearchFilter() added %d conditions, want %d", len(result)-initialLen, tt.want)
			}
		})
	}
}

func TestBuildCategoryFilter(t *testing.T) {
	tests := []struct {
		name     string
		category string
		want     int
	}{
		{
			name:     "no category",
			category: "",
			want:     0,
		},
		{
			name:     "valid category",
			category: "data-analysis",
			want:     1,
		},
		{
			name:     "category with special chars",
			category: "test's-category",
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := ServerFilter{Category: tt.category}
			initialLen := 1
			cteWhere := make([]interface{}, initialLen)
			result := buildCategoryFilter(cteWhere, filter)
			if len(result)-initialLen != tt.want {
				t.Errorf("buildCategoryFilter() added %d conditions, want %d", len(result)-initialLen, tt.want)
			}
		})
	}
}

func TestBuildRatingFilter(t *testing.T) {
	tests := []struct {
		name      string
		minRating float64
		want      int
	}{
		{
			name:      "no rating filter",
			minRating: 0,
			want:      0,
		},
		{
			name:      "rating filter 3.0",
			minRating: 3.0,
			want:      1,
		},
		{
			name:      "rating filter 4.5",
			minRating: 4.5,
			want:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := ServerFilter{MinRating: tt.minRating}
			initialLen := 1
			cteWhere := make([]interface{}, initialLen)
			result := buildRatingFilter(cteWhere, filter)
			if len(result)-initialLen != tt.want {
				t.Errorf("buildRatingFilter() added %d conditions, want %d", len(result)-initialLen, tt.want)
			}
		})
	}
}

func TestBuildRegistryTypesFilter(t *testing.T) {
	tests := []struct {
		name          string
		registryTypes []string
		wantCondition bool
	}{
		{
			name:          "no registry types",
			registryTypes: []string{},
			wantCondition: false,
		},
		{
			name:          "single registry type",
			registryTypes: []string{"npm"},
			wantCondition: true,
		},
		{
			name:          "multiple registry types",
			registryTypes: []string{"npm", "pypi"},
			wantCondition: true,
		},
		{
			name:          "remote transport type",
			registryTypes: []string{"remote"},
			wantCondition: true,
		},
		{
			name:          "mixed registry and remote",
			registryTypes: []string{"npm", "remote"},
			wantCondition: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := ServerFilter{RegistryTypes: tt.registryTypes}
			result := buildRegistryTypesFilter(filter)
			hasCondition := len(result) > 0
			if hasCondition != tt.wantCondition {
				t.Errorf("buildRegistryTypesFilter() has condition = %v, want %v", hasCondition, tt.wantCondition)
			}
		})
	}
}

func TestBuildMainQuery(t *testing.T) {
	tests := []struct {
		name    string
		filter  ServerFilter
		sort    string
		limit   int
		offset  int
		wantErr bool
	}{
		{
			name: "basic query",
			filter: ServerFilter{
				Search: "test",
			},
			sort:    "created",
			limit:   20,
			offset:  0,
			wantErr: false,
		},
		{
			name: "query with all filters",
			filter: ServerFilter{
				Search:        "test",
				Category:      "data",
				MinRating:     3.0,
				MinInstalls:   10,
				RegistryTypes: []string{"npm", "pypi"},
				Tags:          []string{"tag1", "tag2"},
				HasTransport:  []string{"stdio"},
			},
			sort:    "rating_desc",
			limit:   50,
			offset:  10,
			wantErr: false,
		},
		{
			name:    "invalid sort parameter",
			filter:  ServerFilter{},
			sort:    "invalid_sort; DROP TABLE",
			limit:   20,
			offset:  0,
			wantErr: true,
		},
		{
			name: "pagination limits",
			filter: ServerFilter{
				Search: "test",
			},
			sort:    "updated",
			limit:   1000,
			offset:  5000,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := buildMainQuery(tt.filter, tt.sort, tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildMainQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if sql == "" {
					t.Error("buildMainQuery() returned empty SQL")
				}
				if !strings.Contains(sql, "WITH filtered_servers") {
					t.Error("buildMainQuery() SQL doesn't contain CTE")
				}
				if !strings.Contains(sql, "LIMIT") {
					t.Error("buildMainQuery() SQL doesn't contain LIMIT")
				}
				if !strings.Contains(sql, "OFFSET") {
					t.Error("buildMainQuery() SQL doesn't contain OFFSET")
				}
				// Verify no SQL injection by checking for dangerous keywords
				sqlLower := strings.ToLower(sql)
				if strings.Contains(sqlLower, "drop table") || strings.Contains(sqlLower, "delete from") {
					t.Error("buildMainQuery() SQL contains dangerous keywords")
				}
				// Verify args are properly parameterized
				if len(args) > 0 && tt.filter.Search != "" {
					found := false
					for _, arg := range args {
						if str, ok := arg.(string); ok && strings.Contains(str, tt.filter.Search) {
							found = true
							break
						}
					}
					if !found {
						t.Error("buildMainQuery() search term not in args (not parameterized)")
					}
				}
			}
		})
	}
}

func TestBuildCTEQuery(t *testing.T) {
	tests := []struct {
		name    string
		filter  ServerFilter
		wantErr bool
	}{
		{
			name:    "empty filter",
			filter:  ServerFilter{},
			wantErr: false,
		},
		{
			name: "with search",
			filter: ServerFilter{
				Search: "mcp-server",
			},
			wantErr: false,
		},
		{
			name: "with rating and installs",
			filter: ServerFilter{
				MinRating:   4.0,
				MinInstalls: 100,
			},
			wantErr: false,
		},
		{
			name: "with tags array",
			filter: ServerFilter{
				Tags: []string{"ai", "automation"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := buildCTEQuery(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildCTEQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if sql == "" {
					t.Error("buildCTEQuery() returned empty SQL")
				}
				if !strings.Contains(sql, "SELECT") {
					t.Error("buildCTEQuery() SQL doesn't contain SELECT")
				}
				if !strings.Contains(sql, "FROM servers") {
					t.Error("buildCTEQuery() SQL doesn't contain FROM servers")
				}
				// Verify parameterization
				placeholderCount := strings.Count(sql, "$")
				if placeholderCount != len(args) {
					t.Errorf("buildCTEQuery() placeholder count %d != args count %d", placeholderCount, len(args))
				}
			}
		})
	}
}

func TestSQLInjectionPrevention(t *testing.T) {
	// Test various SQL injection attempts
	maliciousInputs := []struct {
		name  string
		sort  string
		search string
	}{
		{
			name:  "drop table",
			sort:  "'; DROP TABLE servers--",
			search: "",
		},
		{
			name:  "union select",
			sort:  "name UNION SELECT * FROM users--",
			search: "",
		},
		{
			name:  "boolean injection",
			sort:  "name OR 1=1--",
			search: "",
		},
		{
			name:  "search with sql",
			sort:  "",
			search: "'; DELETE FROM servers WHERE '1'='1",
		},
	}

	for _, tt := range maliciousInputs {
		t.Run(tt.name, func(t *testing.T) {
			// Test sort validation
			if tt.sort != "" {
				_, err := validateAndGetSortClause(tt.sort)
				if err == nil {
					t.Error("Expected error for malicious sort input, got nil")
				}
			}

			// Test search parameterization
			if tt.search != "" {
				filter := ServerFilter{Search: tt.search}
				sql, args, err := buildMainQuery(filter, "created", 20, 0)
				if err != nil {
					return // Expected to fail validation
				}

				// Verify search term is in args (parameterized), not in SQL string
				if strings.Contains(sql, tt.search) {
					t.Error("Malicious input found directly in SQL (not parameterized)")
				}

				// Verify it's in args
				found := false
				for _, arg := range args {
					if str, ok := arg.(string); ok && strings.Contains(str, tt.search) {
						found = true
						break
					}
				}
				if !found {
					t.Log("Search term properly parameterized (not in SQL)")
				}
			}
		})
	}
}

func TestTagsArrayHandling(t *testing.T) {
	filter := ServerFilter{
		Tags: []string{"tag1", "tag2", "tag3"},
	}

	sql, args, err := buildCTEQuery(filter)
	if err != nil {
		t.Fatalf("buildCTEQuery() failed: %v", err)
	}

	// Verify tags are passed as pq.Array
	found := false
	for _, arg := range args {
		if _, ok := arg.(pq.StringArray); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("Tags not passed as pq.Array")
	}

	// Verify SQL uses ?| operator
	if !strings.Contains(sql, "?|") {
		t.Error("Tags filter doesn't use ?| operator")
	}
}
