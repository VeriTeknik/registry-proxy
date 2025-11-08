# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Important: Repository Structure

This is the main infrastructure repository containing:
- `/registry/` - Submodule pointing to https://github.com/VeriTeknik/registry (upstream MCP Registry)
- `/proxy/` - Enhanced proxy service that adds filtering/sorting capabilities
- `/main/` - Infrastructure configuration (Traefik, PostgreSQL, scripts)

## Project Overview

The plugged.in MCP Registry infrastructure provides:
1. **Core Registry** (submodule) - The official MCP Registry from modelcontextprotocol
2. **Enhanced Proxy** - Adds filtering, sorting, and enriched responses
3. **Infrastructure** - Traefik reverse proxy, PostgreSQL database, deployment scripts

## Architecture

```
User Request → registry.plugged.in → Proxy Service → Core Registry → PostgreSQL
                                          ↓
                                    Enhanced Features:
                                    - Filtering by registry_name
                                    - Sorting by release_date
                                    - Search functionality
                                    - Enriched responses with packages
```

## Key Features Added by Proxy

1. **Enriched List Responses**: `/v0/servers` now includes package information
2. **Filtering**: `?registry_name=npm` to filter by package registry
3. **Sorting**: `?sort=release_date_desc` for various sort orders
4. **Search**: `?search=term` to search in name/description
5. **Caching**: 5-minute cache for improved performance
6. **100% Backward Compatible**: All original endpoints work unchanged

## API Endpoints

### `/v0/servers` (Standard Endpoint)
**Purpose**: Registry-compatible endpoint with caching for performance

**Features**:
- ✅ 5-minute in-memory cache
- ✅ Includes remote headers (e.g., OAuth tokens for Smithery servers)
- ✅ Basic filtering: `?registry_name=npm`, `?search=term`, `?sort=created`
- ✅ Returns structured `EnrichedServer` objects
- ✅ Includes ratings/stats from proxy database

**Use Cases**:
- Standard registry queries
- When you want cached responses for better performance
- Compatible with MCP clients expecting standard registry format

**Response Format**:
```json
{
  "servers": [...],
  "metadata": {
    "count": 10,
    "total": 1609,
    "cached_at": "2025-11-08T04:08:59Z"
  }
}
```

### `/v0/enhanced/servers` (Advanced Endpoint)
**Purpose**: Direct database queries with advanced filtering (added Oct 28, 2025)

**Features**:
- ✅ NO caching - always fresh from database
- ✅ Includes remote headers (e.g., OAuth tokens for Smithery servers)
- ✅ Advanced filtering: `?category=`, `?tags=`, `?min_rating=`, `?registry_types=`
- ✅ Returns raw JSONB from database
- ✅ Includes ratings/stats from proxy database

**Use Cases**:
- When you need guaranteed fresh data
- Advanced filtering requirements
- Debugging cache issues
- Aggregate statistics

**Response Format**:
```json
{
  "servers": [...],
  "total_count": 1609,
  "filters": {...},
  "sort": "created"
}
```

### Ratings & Stats Endpoints
All ratings endpoints work through `/v0/servers/*` paths:
- `POST /v0/servers/{id}/rate` - Submit rating (requires auth)
- `POST /v0/servers/{id}/install` - Track installation (requires auth)
- `GET /v0/servers/{id}/stats` - Get server statistics
- `GET /v0/servers/{id}/reviews` - Get user reviews
- `GET /v0/servers/{id}/feedback` - Get paginated feedback

### When to Use Which Endpoint

| Need | Use |
|------|-----|
| Standard registry queries | `/v0/servers` |
| Best performance (caching) | `/v0/servers` |
| Advanced filtering (tags, ratings) | `/v0/enhanced/servers` |
| Fresh data (no cache) | `/v0/enhanced/servers` |
| Submit ratings/installs | `/v0/servers/{id}/rate` or `/install` |
| Aggregate statistics | `/v0/enhanced/stats/*` |

## Remote Headers Support

**Background**: Many MCP servers (especially Smithery servers) require authentication headers. As of November 8, 2025, the proxy correctly extracts and returns these headers.

**What's Included**:
- Header name (e.g., "Authorization")
- Header value template (e.g., "Bearer {smithery_api_key}")
- Description of what the header is for
- Flags: `isRequired`, `isSecret`

**Example** (from `ai.smithery/zwldarren-akshare-one-mcp`):
```json
"remotes": [{
  "transport_type": "streamable-http",
  "url": "https://server.smithery.ai/@zwldarren/akshare-one-mcp/mcp",
  "headers": [{
    "name": "Authorization",
    "value": "Bearer {smithery_api_key}",
    "description": "Bearer token for Smithery authentication",
    "isRequired": true,
    "isSecret": true
  }]
}]
```

**Database Status**:
- ✅ 1,609 servers synced from official registry (as of Nov 8, 2025)
- ✅ 245 servers have remote headers
- ✅ Schema validation accepts 2025-10-17, 2025-09-29, 2025-09-16

## Troubleshooting

### Headers Not Showing in `/v0/servers`
If headers are missing from `/v0/servers` but present in `/v0/enhanced/servers`:

1. **Clear the cache**:
   ```bash
   curl -X POST https://registry.plugged.in/v0/cache/refresh
   ```

2. **Verify headers in database**:
   ```bash
   docker exec postgresql psql -U mcpregistry -d mcp_registry -c \
     "SELECT value->'remotes'->0->'headers' FROM servers WHERE server_name = 'ai.smithery/zwldarren-akshare-one-mcp' AND is_latest = true;"
   ```

3. **Restart proxy** (if cache refresh doesn't work):
   ```bash
   docker restart registry-proxy
   ```

### Registry Resync After Upstream Updates
When official MCP Registry has updates:

1. **Purge database and resync**:
   ```bash
   # Stop services
   cd /home/pluggedin/registry/main
   ./stop-all.sh

   # Drop and recreate database
   docker start postgresql
   docker exec postgresql psql -U mcpregistry -d postgres -c "DROP DATABASE IF EXISTS mcp_registry;"
   docker exec postgresql psql -U mcpregistry -d postgres -c "CREATE DATABASE mcp_registry OWNER mcpregistry;"

   # Enable seed import in registry/.env
   # MCP_REGISTRY_SEED_FROM=https://registry.modelcontextprotocol.io/v0/servers

   # Restart all services
   ./start-all.sh
   ```

## Common Commands

### Start All Services
```bash
cd /home/pluggedin/registry/main
./start-all.sh
```

### Stop All Services
```bash
cd /home/pluggedin/registry/main
./stop-all.sh
```

### Update Registry Submodule
```bash
cd /home/pluggedin/registry/registry
git pull origin main
cd ..
git add registry
git commit -m "Update registry submodule"
git push
```

### Rebuild Proxy
```bash
cd /home/pluggedin/registry/proxy
docker build -t registry-proxy:v3 .
docker restart registry-proxy
```

### View Logs
```bash
docker logs -f registry-proxy  # Proxy logs
docker logs -f registry        # Core registry logs
docker logs -f postgresql      # Database logs
```

### Test Endpoints
```bash
# Basic health check
curl https://registry.plugged.in/v0/health

# Test filtering
curl "https://registry.plugged.in/v0/servers?registry_name=npm"

# Test sorting
curl "https://registry.plugged.in/v0/servers?sort=release_date_desc"

# Test search
curl "https://registry.plugged.in/v0/servers?search=mcp"
```

## Important Configuration

### Environment Variables

#### Registry Service (.env)
```bash
MCP_REGISTRY_DATABASE_URL=postgres://mcpregistry:password@postgresql:5432/mcp_registry
MCP_REGISTRY_ENVIRONMENT=production
MCP_REGISTRY_SEED_IMPORT=true
MCP_REGISTRY_GITHUB_CLIENT_ID=<your_github_client_id>
MCP_REGISTRY_GITHUB_CLIENT_SECRET=<your_github_client_secret>
```

#### Proxy Service (docker-compose)
```yaml
environment:
  - PROXY_PORT=8090
  - REGISTRY_URL=http://registry:8080
```

### GitHub Secrets for CI/CD
- `DEPLOY_HOST`: Your server IP/hostname
- `DEPLOY_USER`: SSH username
- `DEPLOY_PORT`: SSH port (default 22)
- `DEPLOY_SSH_KEY`: Private SSH key for deployment

## Deployment Notes

1. **Proxy as Main Interface**: All external traffic goes through the proxy
2. **Registry Internal Only**: Core registry not exposed publicly
3. **Traefik Routing**: Handles SSL and routes registry.plugged.in to proxy
4. **PostgreSQL Database**: All services use the same PostgreSQL instance

## Security Considerations

- GitHub OAuth credentials stored in .env (not in git)
- PostgreSQL not exposed externally (accessible only via Docker network)
- All traffic SSL/TLS encrypted via Traefik
- Registry publish endpoint requires valid GitHub token

## Maintenance

- Registry submodule should be kept up to date with upstream
- Proxy can be modified independently without affecting upstream sync
- Database backups recommended before major updates