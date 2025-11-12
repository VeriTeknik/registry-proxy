package db

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/veriteknik/registry-proxy/internal/utils"
	"go.uber.org/zap"
)

var (
	// psql is the PostgreSQL placeholder format for squirrel
	psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// validSortOptions defines the whitelist of allowed sort parameters
	// This prevents SQL injection in ORDER BY clauses
	validSortOptions = map[string]string{
		"":              "published_at DESC",
		"created":       "published_at DESC",
		"name_asc":      "server_name ASC",
		"name_desc":     "server_name DESC",
		"updated":       "updated_at DESC",
		"rating_desc":   "rating DESC, rating_count DESC",
		"reviews_desc":  "rating_count DESC",
		"installs_desc": "installation_count DESC",
		"trending": `(
			installation_count * 0.3 +
			rating_count * 0.3 +
			rating * 10 +
			CASE
				WHEN updated_at > NOW() - INTERVAL '7 days' THEN 20
				WHEN updated_at > NOW() - INTERVAL '30 days' THEN 10
				ELSE 0
			END
		) DESC`,
	}
)

// buildBaseQuery creates the base SELECT statement with CTE for filtered servers
func buildBaseQuery() sq.SelectBuilder {
	return psql.
		Select(
			"server_name",
			"value",
			"published_at",
			"updated_at",
			"rating",
			"rating_count",
			"installation_count",
			"COUNT(*) OVER() as total_count",
		).
		From("filtered_servers")
}

// buildSearchFilter adds search filtering to the query
func buildSearchFilter(cteWhere sq.And, filter ServerFilter) sq.And {
	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		cteWhere = append(cteWhere, sq.Or{
			sq.ILike{"s.server_name": searchTerm},
			sq.Expr("s.value->>'description' ILIKE ?", searchTerm),
		})
	}
	return cteWhere
}

// buildCategoryFilter adds category filtering to the query
func buildCategoryFilter(cteWhere sq.And, filter ServerFilter) sq.And {
	if filter.Category != "" {
		cteWhere = append(cteWhere, sq.Expr("s.value->>'category' = ?", filter.Category))
	}
	return cteWhere
}

// buildTagsFilter adds tags filtering to the query
func buildTagsFilter(cteWhere sq.And, filter ServerFilter) sq.And {
	if len(filter.Tags) > 0 {
		// Use ?| operator for array overlap (PostgreSQL)
		// Note: ?? escapes to single ? in Squirrel, so ??| becomes ?| operator
		cteWhere = append(cteWhere, sq.Expr("s.value->'tags' ??| ?", pq.Array(filter.Tags)))
	}
	return cteWhere
}

// buildRatingFilter adds minimum rating filtering to the query
func buildRatingFilter(cteWhere sq.And, filter ServerFilter) sq.And {
	if filter.MinRating > 0 {
		cteWhere = append(cteWhere, sq.GtOrEq{"COALESCE(ss.rating, 0)": filter.MinRating})
	}
	return cteWhere
}

// buildInstallsFilter adds minimum installs filtering to the query
func buildInstallsFilter(cteWhere sq.And, filter ServerFilter) sq.And {
	if filter.MinInstalls > 0 {
		cteWhere = append(cteWhere, sq.GtOrEq{"COALESCE(ss.installation_count, 0)": filter.MinInstalls})
	}
	return cteWhere
}

// buildRegistryTypesFilter adds registry type filtering to the main query
// Returns an sq.And that should be applied to the main SELECT (not the CTE)
func buildRegistryTypesFilter(filter ServerFilter) sq.And {
	mainWhere := sq.And{}

	if len(filter.RegistryTypes) == 0 {
		return mainWhere
	}

	// Check if "remote" is in the filter
	hasRemote := false
	nonRemoteTypes := []string{}
	for _, rt := range filter.RegistryTypes {
		if rt == "remote" {
			hasRemote = true
		} else {
			nonRemoteTypes = append(nonRemoteTypes, rt)
		}
	}

	// Build conditions based on what we have
	if len(nonRemoteTypes) > 0 && hasRemote {
		// Both non-remote and remote - construct OR manually as single expression
		// Build placeholders for IN clause
		placeholders := make([]string, len(nonRemoteTypes))
		args := make([]interface{}, len(nonRemoteTypes))
		for i, rt := range nonRemoteTypes {
			placeholders[i] = "?"
			args[i] = rt
		}
		sql := fmt.Sprintf(`(
			EXISTS (
				SELECT 1 FROM jsonb_array_elements(value->'packages') p
				WHERE p->>'registryType' IN (%s)
			) OR EXISTS (
				SELECT 1 FROM jsonb_array_elements(value->'remotes') r
				WHERE r->>'type' IN ('sse', 'http', 'streamable-http')
			)
		)`, strings.Join(placeholders, ","))
		mainWhere = append(mainWhere, sq.Expr(sql, args...))
	} else if len(nonRemoteTypes) > 0 {
		// Only non-remote types - use IN instead of ANY
		placeholders := make([]string, len(nonRemoteTypes))
		args := make([]interface{}, len(nonRemoteTypes))
		for i, rt := range nonRemoteTypes {
			placeholders[i] = "?"
			args[i] = rt
		}
		sql := fmt.Sprintf(`EXISTS (
			SELECT 1 FROM jsonb_array_elements(value->'packages') p
			WHERE p->>'registryType' IN (%s)
		)`, strings.Join(placeholders, ","))
		mainWhere = append(mainWhere, sq.Expr(sql, args...))
	} else if hasRemote {
		// Only remote
		mainWhere = append(mainWhere, sq.Expr(
			`EXISTS (
				SELECT 1 FROM jsonb_array_elements(value->'remotes') r
				WHERE r->>'type' IN ('sse', 'http', 'streamable-http')
			)`,
		))
	}

	return mainWhere
}

// buildTransportFilter adds transport type filtering to the main query
func buildTransportFilter(filter ServerFilter) sq.And {
	mainWhere := sq.And{}

	if len(filter.HasTransport) > 0 {
		// Build placeholders for IN clause
		placeholders := make([]string, len(filter.HasTransport))
		args := make([]interface{}, len(filter.HasTransport))
		for i, transport := range filter.HasTransport {
			placeholders[i] = "?"
			args[i] = transport
		}
		sql := fmt.Sprintf(`EXISTS (
			SELECT 1 FROM jsonb_array_elements(value->'packages') p
			WHERE p->'transport'->>'type' IN (%s)
		)`, strings.Join(placeholders, ","))
		mainWhere = append(mainWhere, sq.Expr(sql, args...))
	}

	return mainWhere
}

// validateAndGetSortClause validates the sort parameter and returns the SQL ORDER BY clause
// This prevents SQL injection by using a whitelist approach
func validateAndGetSortClause(sort string) (string, error) {
	sortClause, exists := validSortOptions[sort]
	if !exists {
		// Return available options for better error messages
		validKeys := make([]string, 0, len(validSortOptions))
		for key := range validSortOptions {
			if key != "" { // Skip empty default
				validKeys = append(validKeys, key)
			}
		}
		return "", fmt.Errorf("invalid sort parameter '%s'. Valid options: %s",
			sort, strings.Join(validKeys, ", "))
	}
	return sortClause, nil
}

// buildCTEQuery builds the Common Table Expression (CTE) for filtering servers
func buildCTEQuery(filter ServerFilter) (string, []interface{}, error) {
	// Start with base WHERE clause
	cteWhere := sq.And{sq.Eq{"s.is_latest": true}}

	// Apply all CTE-level filters
	cteWhere = buildSearchFilter(cteWhere, filter)
	cteWhere = buildCategoryFilter(cteWhere, filter)
	cteWhere = buildTagsFilter(cteWhere, filter)
	cteWhere = buildRatingFilter(cteWhere, filter)
	cteWhere = buildInstallsFilter(cteWhere, filter)

	// Build the CTE SELECT statement
	cteSelect := psql.
		Select(
			"s.server_name",
			"s.value",
			"s.published_at",
			"s.updated_at",
			"COALESCE(ss.rating, 0) as rating",
			"COALESCE(ss.rating_count, 0) as rating_count",
			"COALESCE(ss.installation_count, 0) as installation_count",
		).
		From("servers s").
		LeftJoin("proxy_server_stats ss ON s.server_name = ss.server_id").
		Where(cteWhere)

	return cteSelect.ToSql()
}

// buildMainQuery builds the complete query with CTE, main SELECT, filtering, sorting, and pagination
func buildMainQuery(filter ServerFilter, sort string, limit, offset int) (string, []interface{}, error) {
	// Build the CTE
	cteSQL, cteArgs, err := buildCTEQuery(filter)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build CTE: %w", err)
	}

	// Build main SELECT with WHERE clauses for registry types and transports
	mainSelect := buildBaseQuery()

	// Apply registry type and transport filters
	mainWhere := sq.And{}
	mainWhere = append(mainWhere, buildRegistryTypesFilter(filter)...)
	mainWhere = append(mainWhere, buildTransportFilter(filter)...)

	if len(mainWhere) > 0 {
		mainSelect = mainSelect.Where(mainWhere)
	}

	// Validate and add sorting
	sortClause, err := validateAndGetSortClause(sort)
	if err != nil {
		return "", nil, err
	}
	mainSelect = mainSelect.OrderBy(sortClause)

	// Add pagination
	mainSelect = mainSelect.Limit(uint64(limit)).Offset(uint64(offset))

	// Get main SQL and args
	mainSQL, mainArgs, err := mainSelect.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build main query: %w", err)
	}

	// Debug logging (temporary)
	utils.Logger.Error("DEBUG QUERY", zap.String("sql", mainSQL), zap.Any("args", mainArgs))

	// Renumber placeholders in mainSQL to account for CTE args
	// If CTE has N args, main query placeholders should start at $N+1
	placeholderOffset := len(cteArgs)
	renumberedMainSQL := renumberPlaceholders(mainSQL, placeholderOffset)

	// Combine CTE and main query
	// We need to merge the arguments from both parts
	fullSQL := fmt.Sprintf("WITH filtered_servers AS (%s) %s", cteSQL, renumberedMainSQL)
	allArgs := append(cteArgs, mainArgs...)

	return fullSQL, allArgs, nil
}

// renumberPlaceholders renumbers PostgreSQL placeholders ($1, $2, etc.) by adding an offset
// Uses regex with word boundaries to prevent substring matching (e.g., $1 in $10)
func renumberPlaceholders(sql string, offset int) string {
	if offset == 0 {
		return sql
	}

	// Use regex to match whole placeholders with word boundary
	// \b ensures $1 doesn't match inside $10, $11, etc.
	re := regexp.MustCompile(`\$(\d+)\b`)

	return re.ReplaceAllStringFunc(sql, func(match string) string {
		// Extract the number from $N
		num, err := strconv.Atoi(match[1:])
		if err != nil {
			return match // Keep original if parsing fails
		}
		return fmt.Sprintf("$%d", num+offset)
	})
}
