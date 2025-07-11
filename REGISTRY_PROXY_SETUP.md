# Registry Proxy Setup Documentation

## Overview

This document describes the enhanced MCP Registry setup at plugged.in, which uses a proxy service to add filtering, sorting, and enriched data capabilities while maintaining 100% backward compatibility with the original registry API.

## Architecture

```
┌─────────────────┐
│   Internet      │
└────────┬────────┘
         │
    ┌────▼────┐
    │ Traefik │ (SSL, Routing)
    └────┬────┘
         │
         ├─► registry.plugged.in
         │
    ┌────▼─────────────────┐
    │   Registry Proxy      │ (Enhanced API)
    │  - Filtering          │
    │  - Sorting            │
    │  - Enriched responses  │
    └──────────┬────────────┘
               │
    ┌──────────▼────────────┐
    │  Original Registry    │ (Core API)
    │  - GitHub auth        │
    │  - Publishing         │
    │  - Data storage       │
    └──────────┬────────────┘
               │
         ┌─────▼─────┐
         │  MongoDB  │
         └───────────┘
```

## Services

### 1. Registry Proxy (`/home/pluggedin/registry/proxy`)

**Purpose**: Enhances the registry API with additional features while maintaining backward compatibility.

**Features**:
- Enriches `/v0/servers` responses with package data
- Adds filtering by `registry_name` (npm, pip, docker, etc.)
- Adds sorting by `release_date` and `name`
- Adds search functionality
- 5-minute caching for performance
- Passes through all other endpoints unchanged

**Configuration**:
- Port: 8090
- Upstream: http://registry:8080
- Domain: registry.plugged.in (via Traefik)

### 2. Original Registry (`/home/pluggedin/registry/registry`)

**Purpose**: Core MCP registry from upstream repository.

**Features**:
- Server publishing with GitHub authentication
- Server listing and details
- MongoDB data persistence
- Seed data import

**Configuration**:
- Port: 8080 (internal only)
- Database: MongoDB
- GitHub OAuth configured

### 3. MongoDB

**Purpose**: Data persistence for registry.

**Configuration**:
- Port: 27017
- Database: mcp-registry
- Collection: servers_v2

## API Endpoints

All endpoints are available at https://registry.plugged.in

### Enhanced Endpoints

#### GET /v0/servers
Now includes package data and supports:
- `?registry_name=npm` - Filter by package registry
- `?sort=release_date_desc` - Sort options:
  - `release_date_desc` or `newest` (default)
  - `release_date_asc`
  - `name_asc` or `alphabetical`
  - `name_desc`
- `?search=term` - Search in name and description
- `?limit=50` - Results per page (max 500)
- `?offset=0` - Pagination offset

### Pass-through Endpoints

These work exactly as in the original registry:
- GET /v0/health
- GET /v0/servers/{id}
- POST /v0/publish
- GET /v0/ping

## Environment Configuration

### Registry (.env)
```bash
MCP_REGISTRY_DATABASE_URL=mongodb://mongodb:27017
MCP_REGISTRY_ENVIRONMENT=production
MCP_REGISTRY_SEED_IMPORT=true
MCP_REGISTRY_GITHUB_CLIENT_ID=Ov23liauuJvy6sLzrDdr
MCP_REGISTRY_GITHUB_CLIENT_SECRET=<secret>
```

### Proxy (docker-compose)
```yaml
environment:
  - PROXY_PORT=8090
  - REGISTRY_URL=http://registry:8080
```

## Deployment

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

### Update Registry Code
```bash
cd /home/pluggedin/registry/registry
git pull origin main
docker build -t registry:upstream .
docker compose -f docker-compose.prod.yml restart
```

### Update Proxy Code
```bash
cd /home/pluggedin/registry/proxy
# Make changes
docker build -t registry-proxy:v2 .
docker compose -f docker-compose-replace.yml restart
```

## CI/CD Pipeline

GitHub Actions workflows are configured for automatic deployment:

1. **On push to main**:
   - Runs tests
   - Builds Docker images
   - Deploys to production
   - Verifies deployment

2. **Required GitHub Secrets**:
   - DEPLOY_HOST
   - DEPLOY_USER
   - DEPLOY_SSH_KEY
   - DEPLOY_PORT

## Monitoring

### Health Checks
- Registry Proxy: https://registry.plugged.in/health
- Registry Health: https://registry.plugged.in/v0/health

### Logs
```bash
docker logs registry-proxy
docker logs registry
docker logs mongodb
```

### Cache Status
The cache can be manually refreshed:
```bash
curl -X POST https://registry.plugged.in/v0/cache/refresh
```

## Troubleshooting

### Proxy Not Working
1. Check container status: `docker ps`
2. Check logs: `docker logs registry-proxy`
3. Verify upstream registry: `curl http://registry:8080/v0/health`

### Filtering Not Working
1. Check cache: May need refresh
2. Verify package data exists in MongoDB
3. Check proxy logs for errors

### Publishing Fails
1. Verify GitHub credentials in .env
2. Check registry logs: `docker logs registry`
3. Ensure token has proper permissions

## Security Notes

- GitHub OAuth credentials are stored in .env (not in version control)
- Registry is only accessible internally
- All external traffic goes through proxy
- SSL/TLS handled by Traefik

## Future Improvements

- [ ] Add rate limiting
- [ ] Implement webhook notifications
- [ ] Add GraphQL endpoint
- [ ] Implement full-text search with Elasticsearch
- [ ] Add analytics tracking

---

Last Updated: January 2025