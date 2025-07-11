# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Important: Repository Structure

This is the main infrastructure repository containing:
- `/registry/` - Submodule pointing to https://github.com/VeriTeknik/registry (upstream MCP Registry)
- `/proxy/` - Enhanced proxy service that adds filtering/sorting capabilities
- `/main/` - Infrastructure configuration (Traefik, MongoDB, scripts)

## Project Overview

The plugged.in MCP Registry infrastructure provides:
1. **Core Registry** (submodule) - The official MCP Registry from modelcontextprotocol
2. **Enhanced Proxy** - Adds filtering, sorting, and enriched responses
3. **Infrastructure** - Traefik reverse proxy, MongoDB, deployment scripts

## Architecture

```
User Request → registry.plugged.in → Proxy Service → Core Registry → MongoDB
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
docker build -t registry-proxy:v2 .
docker compose -f docker-compose-replace.yml up -d
```

### View Logs
```bash
docker logs -f registry-proxy  # Proxy logs
docker logs -f registry        # Core registry logs
docker logs -f mongodb         # Database logs
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
MCP_REGISTRY_DATABASE_URL=mongodb://mongodb:27017
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
4. **MongoDB Shared**: Both services use the same MongoDB instance

## Security Considerations

- GitHub OAuth credentials stored in .env (not in git)
- MongoDB not exposed externally
- All traffic SSL/TLS encrypted via Traefik
- Registry publish endpoint requires valid GitHub token

## Maintenance

- Registry submodule should be kept up to date with upstream
- Proxy can be modified independently without affecting upstream sync
- Database backups recommended before major updates