# MCP Registry Admin Interface

A secure, internal-only admin interface for managing MCP servers in the registry.

## Features

- 🔐 JWT-based authentication
- 📝 Full CRUD operations for servers
- 🏷️ Status management (active/deprecated)
- 🔄 **Official Registry Sync** - Sync with registry.modelcontextprotocol.io
- 📥 Batch import from MCP config files
- 📊 Audit logging for all operations
- 🎨 Clean, responsive UI with Tailwind CSS
- 🐳 Docker deployment with Traefik integration
- 🔒 HTTPS-only access with security headers

## Architecture

This admin service is **completely separate** from the main registry:
- Runs as independent Docker container
- Direct PostgreSQL access (no registry API dependency)
- Won't be affected by upstream registry updates
- Separate codebase outside the registry submodule

## Quick Start

### 1. Configure Environment

Copy `.env.example` to `.env` and update:

```bash
cp .env.example .env
```

**Important:** Change the default password!

Generate a new password hash:
```bash
# Using htpasswd (requires apache2-utils)
htpasswd -bnBC 10 "" yourpassword | tr -d ':'

# Or using Go
go run -e 'import "golang.org/x/crypto/bcrypt"; import "fmt"; hash, _ := bcrypt.GenerateFromPassword([]byte("yourpassword"), 10); fmt.Println(string(hash))'
```

### 2. Build and Deploy

```bash
# Quick start with provided script
./start.sh

# Or manually:
docker build -t registry-admin:latest .
docker compose -f docker-compose.prod.yml up -d

# Check logs
docker logs registry-admin
```

### 3. Access Admin Interface

Navigate to: `https://admin.registry.plugged.in`

Admin credentials:
- Username: `ckaraca`
- Password: `Helios4victory`

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login with username/password
- `GET /api/auth/verify` - Verify token validity
- `POST /api/auth/logout` - Logout (client-side)

### Server Management
- `GET /api/servers` - List all servers with filters
- `GET /api/servers/:id` - Get server details
- `POST /api/servers` - Create new server
- `PUT /api/servers/:id` - Update server
- `DELETE /api/servers/:id` - Delete server
- `PATCH /api/servers/:id/status` - Update status only
- `POST /api/servers/import` - Batch import servers
- `POST /api/servers/validate` - Validate server configuration

### Registry Sync
- `POST /api/sync/preview` - Preview sync changes (dry run)
- `POST /api/sync/execute` - Execute sync with official registry
- `GET /api/sync/status` - Get last sync status
- `GET /api/sync/history` - View sync history

### Audit
- `GET /api/audit-logs` - View audit trail

## Security

### Authentication
- JWT tokens with 24-hour expiry
- Bcrypt password hashing
- Token required for all API endpoints

### Network Security
- HTTPS-only via Traefik
- HSTS headers enabled
- XSS protection headers
- CORS restricted to admin domain

### Audit Trail
All operations are logged with:
- User identification
- Timestamp
- Action performed
- Server affected
- IP address

## Development

### Local Development

```bash
# Install dependencies
go mod download

# Run locally (requires PostgreSQL)
go run cmd/admin/main.go

# Access at http://localhost:8092
```

### Project Structure

```
admin/
├── cmd/admin/          # Main application entry
├── internal/
│   ├── auth/          # JWT authentication
│   ├── db/            # PostgreSQL operations
│   ├── handlers/      # HTTP handlers
│   ├── middleware/    # Auth & CORS middleware
│   └── models/        # Data models
└── web/static/        # Frontend files
```

## Maintenance

### Backup
Regular PostgreSQL backups recommended:
```bash
pg_dump -h postgresql -U mcpregistry mcp_registry > backup.sql
```

### Update Admin Password
1. Generate new hash (see above)
2. Update `.env` file
3. Restart container: `docker compose restart`

### View Logs
```bash
# Application logs
docker logs registry-admin -f

# Audit logs via UI or PostgreSQL
psql -h postgresql -U mcpregistry -d mcp_registry -c "SELECT * FROM audit_logs ORDER BY timestamp DESC LIMIT 10"
```

## Troubleshooting

### Cannot Login
- Check password hash in `.env`
- Verify JWT_SECRET is set
- Check container logs

### PostgreSQL Connection Failed
- Ensure PostgreSQL is running
- Check network connectivity
- Verify POSTGRES_URL in `.env`

### Traefik Routing Issues
- Check DNS for admin.registry.plugged.in
- Verify Traefik labels in docker-compose.yml
- Check Traefik logs: `docker logs traefik`

## License

Internal use only. Part of the plugged.in MCP Registry infrastructure.