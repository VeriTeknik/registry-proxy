# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the MCP Registry - a community-driven registry service for Model Context Protocol (MCP) servers. It's a Go-based RESTful API that provides centralized discovery and management of MCP server implementations.

## Important: Fork-Specific Development Guidelines

**This is a forked repository that needs to maintain sync capability with upstream.**

### Extension-Only Development Policy

1. **DO NOT modify any upstream files** - Only work in:
   - `/extensions/` directory
   - Fork-specific configuration files (docker-compose overrides, etc.)
   - Fork-specific scripts in `/scripts/` that don't conflict with upstream

2. **All new features must be in `/extensions/`** - This includes:
   - VP (v-plugged) API endpoints
   - Stats and analytics functionality
   - Real-time Redis analytics
   - Any custom handlers or services

3. **Use composition over modification**:
   - Extend existing types rather than modifying them
   - Use interfaces to wrap upstream functionality
   - Create new endpoints rather than modifying existing ones

4. **Configuration extensions**:
   - Use docker-compose override files
   - Add new environment variables with unique prefixes
   - Don't modify existing configuration structures

## Common Development Commands

### Building
```bash
# Docker build (recommended)
docker build -t registry .

# Direct Go build
go build ./cmd/registry

# Build publisher tool
cd tools/publisher && ./build.sh
```

### Running the Service
```bash
# Start with Docker Compose (includes MongoDB)
docker compose up

# Run directly (requires MongoDB running separately)
go run cmd/registry/main.go
```

### Testing
```bash
# Unit tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/...

# Integration tests
./integrationtests/run_tests.sh

# API endpoint tests (requires running server)
./scripts/test_endpoints.sh
```

### Linting
```bash
# Run golangci-lint
golangci-lint run --timeout=5m

# Check formatting
gofmt -s -l .
```

## Architecture Overview

### Core Components

1. **HTTP API Layer** (`internal/api/`)
   - Standard Go net/http server
   - RESTful endpoints: `/v0/health`, `/v0/servers`, `/v0/publish`, etc.
   - Swagger documentation generation
   - Request validation and error handling

2. **Service Layer** (`internal/service/`)
   - Business logic separation from HTTP handlers
   - Database abstraction through interfaces
   - Validation and data transformation

3. **Database Layer** (`internal/database/`)
   - Interface-based design supporting MongoDB and in-memory implementations
   - Repository pattern for data access
   - Automatic seed data import on startup

4. **Authentication** (`internal/auth/`)
   - GitHub OAuth integration for the publish endpoint
   - Bearer token validation
   - User verification against GitHub API

### Key Design Patterns

- **Dependency Injection**: Services receive dependencies through constructors
- **Interface-based Design**: Database and external services use interfaces for testability
- **Context Propagation**: All handlers and services accept context for cancellation/timeouts
- **Error Wrapping**: Consistent error handling with descriptive messages

### API Flow Example (Publish Endpoint)

1. Request hits `/v0/publish` handler in `internal/api/handlers.go`
2. Authentication middleware validates GitHub token
3. Handler parses and validates request body
4. Service layer (`internal/service/publish.go`) processes business logic
5. Database layer persists the server entry
6. Response formatted and returned to client

## Important Conventions

- **Go Module**: Uses Go 1.23 with module `github.com/modelcontextprotocol/registry`
- **Error Handling**: Always wrap errors with context using `fmt.Errorf`
- **Logging**: Use structured logging with appropriate levels
- **Testing**: Unit tests alongside code, integration tests in separate directory
- **API Versioning**: All endpoints prefixed with `/v0`
- **Database Collections**: MongoDB collections versioned (e.g., `servers_v2`)

## Environment Configuration

Key environment variables (prefix: `MCP_REGISTRY_`):
- `DATABASE_URL`: MongoDB connection string
- `SERVER_ADDRESS`: HTTP server bind address
- `GITHUB_CLIENT_ID/SECRET`: GitHub OAuth credentials
- `SEED_IMPORT`: Enable automatic seed data import
- `ENVIRONMENT`: Deployment environment (dev/test/prod)

## Development Tips

- Always run `go mod tidy` after adding dependencies
- Use `docker compose` for consistent development environment
- Check MongoDB indexes when adding new query patterns
- Update Swagger docs when modifying API endpoints
- Integration tests require a running MongoDB instance

## Implementation Strategy: Clean Architecture with Separated Analytics

### Architecture Decision
This fork implements a **clean separation** between Registry (static metadata) and Analytics (dynamic metrics). See `ARCHITECTURE_DECISION.md` for detailed rationale.

### Registry Service (This Repository)
**Focus**: Static server metadata, discovery, and search functionality.

**Minimal Model Extension**:
```go
type ExtendedServer struct {
    model.Server
    Source string `json:"source,omitempty" bson:"source,omitempty"` // "github" | "community" | "private"
}
```

**Core Endpoints** (minimal enhancements):
- `GET /v0/servers` - Basic filtering: `?source=github|community|private&search=term&limit=50&offset=0`
- `POST /v0/publish` - GitHub repos (upstream compatibility)
- `POST /v0/servers/community` - Community submissions
- `POST /v0/servers/private` - Private repo submissions

**Event Emission**: Registry emits events for analytics consumption:
- `server_published`, `server_viewed`, `server_searched`, `server_updated`

### Analytics Service (Separate Service)
**Focus**: All dynamic metrics, usage tracking, trends, and ratings.

**Responsibilities**:
- Installation tracking
- Ratings and reviews
- MCP capability analytics (tools, prompts, resources)
- Transport type metrics
- Real-time statistics
- Elasticsearch integration

### Key Principles
1. **Minimal Registry Changes**: Only add `source` field, keep upstream sync easy
2. **Event-Driven Architecture**: Registry emits, Analytics consumes
3. **No Analytics in Registry**: All metrics handled externally
4. **Technology Freedom**: Analytics can use different stack
5. **Clean Separation**: Each service has single responsibility

### Implementation Guide
See `CLEAN_ARCHITECTURE_GUIDE.md` for detailed implementation instructions.

### Benefits
- Easy upstream syncing (minimal registry changes)
- Independent scaling of services
- Clear separation of concerns
- Registry stays fast and focused
- Analytics can evolve independently