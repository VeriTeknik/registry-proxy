# Plugged.in Registry PostgreSQL Migration & Subregistry Implementation Plan

## Date: October 22, 2024
## Status: Ready for Implementation

---

# Executive Summary

Complete migration from MongoDB to PostgreSQL and transformation into a proper MCP subregistry that:
- Syncs with official registry (registry.modelcontextprotocol.io)
- Adds value through `_meta` fields (stats, ratings, curation)
- Enables major app redesign with enhanced features
- Maintains single PostgreSQL database for all data

---

# Architecture Overview

```
Official MCP Registry → ETL (15min sync) → Plugged.in Subregistry → Pluggedin App
                                                ↓
                                        PostgreSQL Database
                                        - Registry tables (upstream schema)
                                        - Stats tables (ratings, installs)
                                        - User data (collections, etc.)
```

---

# Phase 1: Core Registry Setup

## 1.1 Update Registry to Latest Upstream

```bash
cd /home/pluggedin/registry
git fetch upstream
git checkout upstream/main  # 136 commits ahead with PostgreSQL
```

## 1.2 PostgreSQL Infrastructure

```yaml
services:
  postgresql:
    image: postgres:16-alpine
    container_name: postgresql
    environment:
      POSTGRES_DB: pluggedin_registry
      POSTGRES_USER: pluggedin
      POSTGRES_PASSWORD: [secure-password]
    volumes:
      - ./postgresql/data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U pluggedin -d pluggedin_registry"]
    ports:
      - "5432:5432"
    networks:
      - default

  registry:
    build: ./registry
    container_name: registry
    depends_on:
      postgresql:
        condition: service_healthy
    environment:
      MCP_REGISTRY_DATABASE_URL: postgres://pluggedin:[password]@postgresql:5432/pluggedin_registry
      MCP_REGISTRY_GITHUB_CLIENT_ID: [github-oauth-id]
      MCP_REGISTRY_GITHUB_CLIENT_SECRET: [github-oauth-secret]
      MCP_REGISTRY_SEED_FROM: https://registry.modelcontextprotocol.io
      MCP_REGISTRY_OIDC_ENABLED: true
      MCP_REGISTRY_OIDC_ISSUER: https://accounts.google.com
      MCP_REGISTRY_OIDC_CLIENT_ID: [google-oauth-client-id]
      MCP_REGISTRY_OIDC_EXTRA_CLAIMS: '[{"hd":"plugged.in"}]'
      MCP_REGISTRY_OIDC_EDIT_PERMISSIONS: 'admin@plugged.in'
    networks:
      - default
      - traefik
```

---

# Phase 2: Enhancement Database Schema

## 2.1 Core Registry Tables (from upstream)

```sql
-- Provided by upstream migrations
-- servers, packages, versions, etc.
```

## 2.2 Plugged.in Enhancement Tables

```sql
-- Stats tracking
CREATE TABLE server_stats (
  server_id UUID PRIMARY KEY REFERENCES servers(id),
  installation_count INTEGER DEFAULT 0,
  rating DECIMAL(3,2) DEFAULT 0,
  rating_count INTEGER DEFAULT 0,
  unique_users INTEGER DEFAULT 0,
  last_7_days_installs INTEGER DEFAULT 0,
  last_30_days_installs INTEGER DEFAULT 0,
  trending_score DECIMAL(5,2) DEFAULT 0,
  updated_at TIMESTAMP DEFAULT NOW()
);

-- User interactions
CREATE TABLE user_ratings (
  id SERIAL PRIMARY KEY,
  server_id UUID REFERENCES servers(id),
  user_id UUID NOT NULL,
  rating INTEGER CHECK (rating BETWEEN 1 AND 5),
  review TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(server_id, user_id)
);

CREATE TABLE user_installations (
  id SERIAL PRIMARY KEY,
  server_id UUID REFERENCES servers(id),
  user_id UUID NOT NULL,
  version VARCHAR(50),
  created_at TIMESTAMP DEFAULT NOW()
);

-- Curation/Collections
CREATE TABLE collections (
  id UUID PRIMARY KEY,
  name VARCHAR(255),
  description TEXT,
  owner_id UUID,
  is_public BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE collection_servers (
  collection_id UUID REFERENCES collections(id),
  server_id UUID REFERENCES servers(id),
  added_at TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (collection_id, server_id)
);

-- Indexes for performance
CREATE INDEX idx_server_stats_trending ON server_stats(trending_score DESC);
CREATE INDEX idx_ratings_server ON user_ratings(server_id);
CREATE INDEX idx_installations_server ON user_installations(server_id);
CREATE INDEX idx_installations_user ON user_installations(user_id);
```

---

# Phase 3: Subregistry Implementation

## 3.1 ETL Process

```go
// Sync from official registry every 15 minutes
func syncFromOfficialRegistry() {
    // Fetch all servers
    response, _ := http.Get("https://registry.modelcontextprotocol.io/v0/servers")
    servers := parseServers(response)

    for _, server := range servers {
        // Upsert server maintaining existing stats
        tx := db.Begin()

        // Upsert server record
        db.Exec(`
            INSERT INTO servers (id, name, description, status, version)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (id) DO UPDATE SET
                name = EXCLUDED.name,
                description = EXCLUDED.description,
                status = EXCLUDED.status,
                version = EXCLUDED.version,
                updated_at = NOW()
        `, server.ID, server.Name, server.Description, server.Status, server.Version)

        // Initialize stats if new
        db.Exec(`
            INSERT INTO server_stats (server_id)
            VALUES ($1)
            ON CONFLICT (server_id) DO NOTHING
        `, server.ID)

        tx.Commit()
    }

    // Mark servers not in latest sync as deleted
    db.Exec(`
        UPDATE servers
        SET status = 'deleted'
        WHERE updated_at < NOW() - INTERVAL '30 minutes'
        AND status = 'active'
    `)
}
```

## 3.2 API Response with _meta

```json
{
  "$schema": "https://static.modelcontextprotocol.io/schemas/2025-07-09/server.schema.json",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "io.github.user/weather-server",
  "description": "MCP server for weather data",
  "status": "active",
  "version": "1.0.0",
  "packages": [...],
  "repository": {...},
  "_meta": {
    "in.plugged.registry/analytics": {
      "installation_count": 1543,
      "rating": 4.5,
      "rating_count": 67,
      "unique_users": 892,
      "trending_rank": 5,
      "last_updated": "2025-10-22T10:00:00Z"
    },
    "in.plugged.registry/curation": {
      "featured": true,
      "collections": ["productivity", "ai-tools"],
      "tags": ["weather", "api", "real-time"]
    }
  }
}
```

---

# Phase 4: API Endpoints

## 4.1 Standard MCP Registry Endpoints

- `GET /v0/health` - Health check
- `GET /v0/servers` - List servers with filtering
- `GET /v0/servers/{id}` - Get server details
- `POST /v0/publish` - Publish new server
- `PUT /v0/servers/{id}` - Admin edit (OIDC auth)

## 4.2 Enhanced Endpoints

- `POST /v0/servers/{id}/install` - Track installation
- `POST /v0/servers/{id}/rate` - Submit rating
- `GET /v0/servers/{id}/reviews` - Get reviews
- `GET /v0/trending` - Trending servers
- `GET /v0/featured` - Featured servers
- `GET /v0/collections` - Browse collections
- `GET /v0/search` - Advanced search

## 4.3 Query Parameters

```
/v0/servers?search=keyword&package_type=npm&min_rating=4&sort=trending&tag=productivity
```

---

# Phase 5: App Integration Updates

## 5.1 Updated Registry Client

```typescript
// New simplified client
export class PluggedinRegistryClient {
  private baseUrl = process.env.REGISTRY_API_URL || 'https://registry.plugged.in/v0';

  async listServers(params: ServerParams): Promise<ServersResponse> {
    const response = await fetch(`${this.baseUrl}/servers?${new URLSearchParams(params)}`);
    return response.json();
  }

  async trackInstall(serverId: string, userId: string): Promise<void> {
    await fetch(`${this.baseUrl}/servers/${serverId}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId })
    });
  }

  async submitRating(serverId: string, rating: number, userId: string): Promise<void> {
    await fetch(`${this.baseUrl}/servers/${serverId}/rate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rating, user_id: userId })
    });
  }
}
```

## 5.2 Server Interface with _meta

```typescript
interface MCPServer {
  id: string;
  name: string;
  description: string;
  packages: Package[];
  _meta?: {
    'in.plugged.registry/analytics'?: {
      installation_count: number;
      rating: number;
      rating_count: number;
      trending_rank?: number;
    };
    'in.plugged.registry/curation'?: {
      featured: boolean;
      collections: string[];
      tags: string[];
    };
  };
}
```

---

# Phase 6: Deployment Steps

## 6.1 Pre-deployment

1. Backup current MongoDB data
2. Export any custom server configurations
3. Document current stats/ratings

## 6.2 Deployment Sequence

```bash
# 1. Stop current services
cd /home/pluggedin/registry/main
./stop-all.sh

# 2. Deploy PostgreSQL
docker compose up -d postgresql
docker compose exec postgresql pg_isready

# 3. Build and deploy registry
cd /home/pluggedin/registry/registry
docker build -t registry:latest .
cd ../main
docker compose up -d registry

# 4. Run migrations
docker exec registry ./registry migrate up

# 5. Create enhancement tables
docker exec postgresql psql -U pluggedin -d pluggedin_registry < /sql/enhance.sql

# 6. Initial sync from official registry
docker exec registry ./registry sync

# 7. Import historical stats (if available)
docker exec postgresql psql -U pluggedin -d pluggedin_registry < /backup/stats.sql

# 8. Deploy updated app
cd /home/pluggedin/registry/pluggedin-app
npm run build
docker build -t pluggedin-app:latest .
docker compose up -d app

# 9. Verify all endpoints
./test-endpoints.sh
```

---

# Testing Checklist

## Registry Core
- [ ] GET /v0/health returns ok
- [ ] GET /v0/servers returns server list
- [ ] GET /v0/servers/{id} returns details
- [ ] Servers include _meta fields

## Enhanced Features
- [ ] POST /v0/servers/{id}/install tracks
- [ ] POST /v0/servers/{id}/rate works
- [ ] GET /v0/trending returns results
- [ ] Filtering by package_type works
- [ ] Search functionality works

## App Integration
- [ ] Server list displays
- [ ] Stats show correctly
- [ ] Rating submission works
- [ ] Installation tracking works
- [ ] Search and filters work

---

# Rollback Plan

If issues occur:

```bash
# 1. Stop new services
docker compose down

# 2. Restore MongoDB
cd /home/pluggedin/registry/main
docker compose -f docker-compose.old.yml up -d mongodb

# 3. Restore old registry
docker run -d --name registry-old registry:mongodb-version

# 4. Restore old app
docker run -d --name app-old pluggedin-app:old

# 5. Update Traefik routes
# Point back to old services
```

---

# Timeline Estimate

- PostgreSQL setup: 1 hour
- Registry deployment: 2 hours
- Enhancement tables: 1 hour
- ETL implementation: 3 hours
- API endpoints: 3 hours
- App updates: 4-6 hours
- Testing: 2 hours
- **Total: 16-20 hours**

---

# Future Enhancements

With this architecture, we can add:
- AI-powered recommendations
- Server health monitoring
- Automated testing of servers
- Marketplace features
- Developer API keys
- WebSocket subscriptions

---

# Success Metrics

- [ ] All existing features work
- [ ] Performance improved (faster queries)
- [ ] Stats accurately tracked
- [ ] ETL runs reliably
- [ ] App UX enhanced
- [ ] Admin operations simplified

---

# Notes

- This plan creates a sustainable, scalable platform
- Follows MCP ecosystem best practices
- Positions plugged.in as a premier subregistry
- Enables future growth and features