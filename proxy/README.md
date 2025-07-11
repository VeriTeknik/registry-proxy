# Registry Proxy Service

This proxy service enriches the MCP Registry API responses with package information, enabling frontend filtering and sorting capabilities.

## Features

- **Enriched Data**: Combines server list with package details in a single response
- **Filtering**: Filter servers by package registry name (npm, pip, etc.)
- **Sorting**: Sort by release date (ascending/descending) or name
- **Search**: Search servers by name or description
- **Caching**: 5-minute cache for improved performance
- **Pagination**: Offset-based pagination support

## API Endpoints

### GET /v0/servers

Enhanced server list with package information.

**Query Parameters:**
- `registry_name`: Filter by package registry (e.g., "npm", "pip")
- `sort`: Sort order
  - `release_date_desc` or `newest` (default)
  - `release_date_asc`
  - `name_asc` or `alphabetical`
  - `name_desc`
- `search`: Search term for name/description
- `limit`: Results per page (default: 30, max: 500)
- `offset`: Number of results to skip

**Example Response:**
```json
{
  "servers": [
    {
      "id": "...",
      "name": "io.github.example/server",
      "description": "...",
      "repository": {...},
      "version_detail": {
        "version": "1.0.0",
        "release_date": "2025-01-10T12:00:00Z",
        "is_latest": true
      },
      "packages": [
        {
          "registry_name": "npm",
          "name": "@example/server",
          "version": "1.0.0"
        }
      ]
    }
  ],
  "metadata": {
    "count": 30,
    "total": 396,
    "filtered_by": "npm",
    "sorted_by": "newest",
    "cached_at": "2025-01-10T22:30:00Z"
  }
}
```

### POST /v0/cache/refresh

Force a cache refresh.

## Deployment

### With Docker Compose

```bash
cd /home/pluggedin/registry/proxy
docker compose up -d
```

### Environment Variables

- `PROXY_PORT`: Port to listen on (default: 8090)
- `REGISTRY_URL`: Upstream registry URL (default: http://registry:8080)

## Development

```bash
# Install dependencies
go mod download

# Run locally
go run cmd/proxy/main.go

# Build
go build -o proxy cmd/proxy/main.go
```

## Architecture

The proxy service:
1. Fetches the complete server list from the upstream registry
2. Enriches each server with package details (parallel requests)
3. Caches the enriched data for 5 minutes
4. Applies filters and sorting based on query parameters
5. Returns paginated results

This approach ensures zero modifications to the upstream registry while providing the enhanced functionality needed for frontend manipulation.