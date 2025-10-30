# Security Documentation

This document outlines the security measures implemented in the Registry Proxy.

## SQL Injection Prevention

### Query Builder
All database queries use the [Squirrel](https://github.com/Masterminds/squirrel) query builder, which automatically handles parameterization and prevents SQL injection attacks.

### Sort Parameter Validation
Sort parameters are validated against a whitelist before being used in queries:

```go
validSortOptions = map[string]string{
    "created":       "published_at DESC",
    "name_asc":      "server_name ASC",
    "rating_desc":   "rating DESC",
    // ... more options
}
```

Any sort parameter not in this whitelist will be rejected with an error.

## Input Validation

### Server ID Validation
All server IDs are validated to prevent path traversal and injection attacks:

- Must be 1-255 characters
- Cannot contain `..`, `/`, or `\`
- Must match either UUID format or qualified name format (alphanumeric with `.`, `-`, `_`)

### Request Validation
All request parameters are validated using [go-playground/validator](https://github.com/go-playground/validator):

- Search queries: max 200 characters
- Categories: max 100 characters
- Ratings: 0-5 range
- Registry types: whitelist of valid values (npm, pypi, oci, mcpb, nuget, remote)
- Transports: whitelist of valid values (stdio, sse, http)
- Pagination: limit 1-1000, offset >= 0

## Request Security

### Request Size Limits
All POST/PUT endpoints enforce a maximum request body size of 1MB to prevent DoS attacks via large payloads.

### Request Timeouts
All requests have a 30-second timeout (configurable via `REQUEST_TIMEOUT` environment variable) to prevent resource exhaustion from long-running requests.

## CORS Configuration

CORS is configured based on deployment type:

- **Environment Variable**: `CORS_ALLOWED_ORIGIN`
- **Default**: `*` (public hosted registry - allows all origins)
- **Private Deployments**: Set to specific domain if needed

### Public vs Private Registries

**Public Hosted Registry (Default)**:
- CORS: `*` is **correct and appropriate**
- The registry is designed to be accessed from any website/application
- Users can call the API from their own domains
- Authentication is handled via API keys (not cookies/credentials)

**Private Internal Registry (Optional)**:
- CORS: Set `CORS_ALLOWED_ORIGIN=https://yourinternaldomain.com`
- Only needed for private corporate deployments
- Restricts access to specific domains

**Note**: For public APIs like this registry, `Access-Control-Allow-Origin: *` is the standard and secure approach, as authentication is token-based (not cookie-based).

## Logging Security

### Structured Logging
The proxy uses [zap](https://github.com/uber-go/zap) for structured logging, which prevents log injection attacks.

### No Sensitive Data in Logs
- API keys are NEVER logged
- User passwords are NEVER logged
- Database errors are logged internally but sanitized before returning to clients
- Server IDs in logs are validated strings only

### Log Levels
- **Development**: DEBUG level with console output
- **Production**: INFO level with JSON output

Configure via environment variables:
- `ENVIRONMENT`: "development" or "production"
- `LOG_LEVEL`: "debug", "info", "warn", "error"

## Error Handling

### Generic Error Messages
All error responses to clients use generic messages to prevent information disclosure:

- ✅ "Internal server error"
- ✅ "Invalid server ID"
- ❌ "Database connection failed: timeout connecting to 192.168.1.1"
- ❌ "SQL error: table 'passwords' does not exist"

Detailed errors are logged server-side only.

## Authentication

### API Key Authentication
Write operations (rate, install) require API key authentication:

- Provided via `Authorization` header
- Validated by middleware before handler execution
- Not logged or exposed in responses

## Environment Variables

### Required for Production

```bash
# Database (REQUIRED)
DATABASE_URL=postgres://user:pass@host:5432/dbname

# Environment (REQUIRED)
ENVIRONMENT=production
```

### Optional Configuration

```bash
# CORS - Only set for private deployments
# For public hosted registry: Leave unset (defaults to "*")
# For private internal registry: Set to your domain
# CORS_ALLOWED_ORIGIN=https://yourinternaldomain.com

# Request timeout (default: 30s)
REQUEST_TIMEOUT=30s

# Logging (defaults: production/info in production, development/debug in dev)
LOG_LEVEL=info
```

## Security Testing

### Testing for SQL Injection

```bash
# These should all be rejected:
curl "http://localhost:8090/v0/enhanced/servers?sort=; DROP TABLE servers--"
curl "http://localhost:8090/v0/servers/../../../etc/passwd"
curl "http://localhost:8090/v0/servers/'; DELETE FROM servers WHERE '1'='1"
```

### Testing Input Validation

```bash
# Invalid rating (should return 400)
curl -X POST -H "Authorization: Bearer $KEY" \
  -d '{"rating": 10}' \
  http://localhost:8090/v0/servers/test/rate

# Invalid server ID (should return 400)
curl "http://localhost:8090/v0/servers/../../etc/passwd"

# Oversized request (should return 413)
curl -X POST -H "Authorization: Bearer $KEY" \
  -d "$(head -c 2M /dev/urandom | base64)" \
  http://localhost:8090/v0/servers/test/rate
```

### Testing CORS

```bash
# For public registry (default): Should allow any origin
curl -v -H "Origin: https://example.com" \
  http://localhost:8090/v0/enhanced/servers
# Response should include: Access-Control-Allow-Origin: *

# For private registry (if CORS_ALLOWED_ORIGIN is set): Should only allow configured origin
export CORS_ALLOWED_ORIGIN=https://yourdomain.com
# Restart service, then test:
curl -v -H "Origin: https://yourdomain.com" \
  http://localhost:8090/v0/enhanced/servers
# Response should include: Access-Control-Allow-Origin: https://yourdomain.com

curl -v -H "Origin: https://evil.com" \
  http://localhost:8090/v0/enhanced/servers
# Response should include: Access-Control-Allow-Origin: https://yourdomain.com (not evil.com)
```

## Vulnerability Reporting

If you discover a security vulnerability, please email [security contact] with:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

Please do not open public issues for security vulnerabilities.

## Compliance

This proxy implements security measures aligned with:

- OWASP Top 10 protections
- GDPR data protection principles (no PII in logs)
- SOC 2 logging and monitoring requirements

## Security Checklist for Deployment

- [ ] **Public Registry**: Leave `CORS_ALLOWED_ORIGIN` unset (defaults to `*`) OR **Private Registry**: Set to your internal domain
- [ ] Set `ENVIRONMENT=production`
- [ ] Set `LOG_LEVEL=info` or higher
- [ ] Use strong database passwords (not default values)
- [ ] Enable HTTPS/TLS at load balancer/reverse proxy
- [ ] Set `REQUEST_TIMEOUT` appropriately for your use case (default 30s is usually good)
- [ ] Rotate API keys regularly (for write operations)
- [ ] Monitor logs for suspicious activity (unusual query patterns, injection attempts)
- [ ] Keep dependencies updated (`go get -u && go mod tidy`)
- [ ] Ensure PostgreSQL is not exposed publicly (only internal network access)
- [ ] Set up regular database backups

## Recent Security Improvements

- **2025-01**: Complete refactoring to use Squirrel query builder (eliminates SQL injection)
- **2025-01**: Added comprehensive input validation with go-playground/validator
- **2025-01**: Implemented configurable CORS (defaults to `*` for public registry, optional restriction for private deployments)
- **2025-01**: Added request timeouts (30s default) and size limits (1MB for POST bodies)
- **2025-01**: Migrated to structured logging with zap (prevents log injection, removes sensitive data)
- **2025-01**: Sanitized all error messages (no internal details exposed to clients)
- **2025-01**: Added server ID validation (prevents path traversal attacks)
