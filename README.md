# MCP Registry Proxy

[![Version](https://img.shields.io/badge/version-1.0.0-blue)](https://github.com/VeriTeknik/registry-proxy/releases)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/postgresql-15+-blue)](https://www.postgresql.org/)

## Overview

Enhanced MCP Registry Proxy with PostgreSQL backend, rating system, and admin synchronization with the official MCP Registry at registry.modelcontextprotocol.io.

## 🎯 What's New in v1.0.0

### ✨ Major Features
- **PostgreSQL Migration**: Complete migration from MongoDB to PostgreSQL for better performance and ACID compliance
- **Rating & Review System**: 5-star ratings and written reviews for MCP servers
- **Admin Sync**: Synchronize with official registry.modelcontextprotocol.io
- **Security Hardening**: SQL injection prevention, input validation, authentication middleware
- **Enhanced Admin UI**: Server management, status control, search & filter, pagination
- **Performance Optimizations**: Database indexes, connection pooling, efficient caching

See [CHANGELOG](https://github.com/VeriTeknik/registry-proxy/releases/tag/v1.0.0) for complete release notes.

## Directory Structure

```
.
├── proxy/                   # Main proxy service (Go)
├── admin/                   # Admin web interface (Go + JavaScript)
├── main/                    # Deployment infrastructure
│   ├── docker-compose.yml   # PostgreSQL, Traefik, services
│   └── enhancement_schema.sql # Database schema
├── registry/                # Upstream MCP Registry (submodule)
└── docs/                    # Documentation
```

## Active Services

### 1. Main Infrastructure (`/main`)

The central deployment directory containing:

- **docker-compose.yml**: Core infrastructure services
  - Traefik (reverse proxy with SSL)
  - PostgreSQL (primary database)
  - Proxy Service (Enhanced MCP Registry API)
  - Admin Service (Management interface)
  
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
    │ Traefik │ (SSL, Routing, Load Balancing)
    └────┬────┘
         │
    ┌────▼─────────────────────────────────┐
    │   Proxy Service (Port 8090)          │
    │   • Server listings & search          │
    │   • Rating & review system            │
    │   • Stats & analytics                 │
    │   • Sync with official registry       │
    └────┬──────────────────────────────────┘
         │
    ┌────▼─────────────────────────────────┐
    │   Admin Service (Port 8091)          │
    │   • Server management UI              │
    │   • Sync preview & execution          │
    │   • Status management                 │
    │   • Import/Export tools               │
    └────┬──────────────────────────────────┘
         │
    ┌────▼──────────┐
    │  PostgreSQL   │ (Primary Database)
    │   • servers   │ (Server metadata with versioning)
    │   • server_stats  │ (Ratings, installs, reviews)
    │   • server_ratings │ (Individual user ratings)
    │   • server_reviews │ (User reviews)
    └───────────────┘
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

PostgreSQL is the primary database (migrated from MongoDB in v1.0.0):
- **Container**: postgres
- **Port**: 5432
- **Database**: pluggedin
- **User**: postgres

### Database Schema

**Core Tables:**
- `servers` - Server metadata with versioning support
- `server_stats` - Installation counts, ratings, and analytics
- `server_ratings` - Individual user ratings (1-5 stars)
- `server_reviews` - Written reviews with timestamps

### Accessing PostgreSQL

```bash
# Connect to database
docker exec -it postgres psql -U postgres -d pluggedin

# View servers
SELECT server_name, version, status FROM servers WHERE is_latest = true LIMIT 10;

# View ratings
SELECT server_id, AVG(rating) as avg_rating, COUNT(*) as total_ratings
FROM server_ratings GROUP BY server_id;
```

### Migration from MongoDB

See [POSTGRES_MIGRATION_PLAN.md](POSTGRES_MIGRATION_PLAN.md) for complete migration guide.

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

## 📝 Release Notes

### v1.0.0 - PostgreSQL Migration & Production Ready (2025-10-28)

**Major Features:**
- Complete PostgreSQL migration with optimized schema
- Rating & review system for MCP servers
- Admin sync with official registry
- Comprehensive security hardening
- Enhanced admin interface

**Breaking Changes:**
- MongoDB no longer supported (PostgreSQL required)
- New environment variables required (see .env.example)
- Admin authentication now mandatory

See [full release notes](https://github.com/VeriTeknik/registry-proxy/releases/tag/v1.0.0) for details.

## Contact

For issues or questions:
- **GitHub Issues**: https://github.com/VeriTeknik/registry-proxy/issues
- **Documentation**: See DEPLOYMENT_COMPLETE.md and POSTGRES_MIGRATION_PLAN.md
- **Official Registry**: https://registry.modelcontextprotocol.io

---

**Version**: 1.0.0
**Last Updated**: October 28, 2025
**License**: MIT