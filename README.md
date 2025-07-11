# plugged.in Infrastructure Documentation

## Overview

This directory contains the infrastructure and services for plugged.in, a community platform for Model Context Protocol (MCP) servers.

## Directory Structure

```
/home/pluggedin/
├── main/                    # Main deployment infrastructure
├── registry/                # MCP Registry (upstream fork)
├── mcp-analytics/           # Analytics service (to be deprecated)
└── registry-obsolete/       # Old registry with analytics (archived)
```

## Active Services

### 1. Main Infrastructure (`/main`)

The central deployment directory containing:

- **docker-compose.yml**: Core infrastructure services
  - Traefik (reverse proxy with SSL)
  - MongoDB (shared database)
  
- **Scripts**:
  - `start-all.sh` - Starts all services
  - `stop-all.sh` - Stops all services
  
- **Configuration**:
  - `traefik/acme.json` - Let's Encrypt SSL certificates

### 2. MCP Registry (`/registry`)

Clean fork of the official MCP registry from modelcontextprotocol/registry.

- **Purpose**: Provides the official registry API
- **Modifications**: Minimal, to stay compatible with upstream
- **API**: Available at https://registry.plugged.in

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
    ┌────▼────────────────┐
    │  Registry Service   │
    │  (MCP Servers DB)   │
    └──────────┬──────────┘
               │
         ┌─────▼─────┐
         │  MongoDB  │
         └───────────┘
```

## Service URLs

- **Registry API**: https://registry.plugged.in
- **Traefik Dashboard**: http://localhost:8080 (local only)

## Getting Started

### Starting Services

```bash
cd /home/pluggedin/main
./start-all.sh
```

### Stopping Services

```bash
cd /home/pluggedin/main
./stop-all.sh
```

### Checking Service Status

```bash
docker ps
```

## Database

MongoDB is used as the primary database:
- **Container**: mongodb
- **Port**: 27017
- **Database**: mcp-registry
- **Collection**: servers_v2

### Accessing MongoDB

```bash
docker exec -it mongodb mongosh mcp-registry
```

## SSL Certificates

SSL certificates are managed by Traefik using Let's Encrypt:
- **Location**: `/home/pluggedin/main/traefik/acme.json`
- **Domains**: Automatically provisioned for configured hosts
- **Renewal**: Automatic

## Development Workflow

### Updating Registry

1. Pull upstream changes:
   ```bash
   cd /home/pluggedin/registry
   git fetch upstream
   git merge upstream/main
   ```

2. Rebuild if needed:
   ```bash
   docker compose -f docker-compose.local.yml build
   ```

3. Restart services:
   ```bash
   cd /home/pluggedin/main
   ./stop-all.sh
   ./start-all.sh
   ```

## Troubleshooting

### Services not starting

1. Check Docker logs:
   ```bash
   docker logs <container-name>
   ```

2. Verify networks exist:
   ```bash
   docker network ls
   ```

3. Check port conflicts:
   ```bash
   netstat -tulpn | grep -E '(80|443|8080|27017)'
   ```

### SSL Certificate Issues

1. Check Traefik logs:
   ```bash
   docker logs traefik
   ```

2. Verify DNS points to server
3. Check acme.json permissions (should be 600)

## Future Architecture

The plan is to evolve toward:

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│ plugged.in   │────▶│ Community API   │────▶│ Community DB │
│ Frontend     │     │ (Your Backend)  │     │ (Your Data)  │
└──────┬───────┘     └─────────────────┘     └──────────────┘
       │                                                       
       ▼                                                       
┌──────────────┐                                              
│ MCP Registry │ (Official registry.modelcontextprotocol.io)  
└──────────────┘                                              
```

This separation allows:
- Independent development of community features
- Easy migration when official registry launches
- Clean upstream compatibility

## Deprecated Services

### Analytics Service (`/registry-obsolete/analytics`)

Previously provided search and analytics features. Deprecated due to:
- High maintenance overhead
- Complex infrastructure (Elasticsearch, Kibana, Redis)
- Upstream registry evolving rapidly

Data and code archived for reference.

## Security Notes

- All services run in Docker containers
- MongoDB is not exposed externally (only via Docker network)
- SSL/TLS enforced for all public endpoints
- Traefik dashboard only accessible locally

## Maintenance

### Daily
- Monitor service health
- Check disk space for MongoDB

### Weekly
- Review Docker logs for errors
- Update Docker images if needed

### Monthly
- Review and update dependencies
- Check SSL certificate renewal
- Backup MongoDB data

## Contact

For issues or questions about the infrastructure, refer to the GitHub repository or documentation at https://docs.plugged.in

---

Last Updated: January 2025