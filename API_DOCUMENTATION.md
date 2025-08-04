# MCP Registry API Documentation

## Overview

The MCP Registry API provides a centralized service for discovering and managing Model Context Protocol (MCP) servers. This API is available at `https://registry.plugged.in` and offers enhanced features including filtering, sorting, and search capabilities.

## Base URL

```
https://registry.plugged.in
```

## Authentication

Most endpoints are publicly accessible. Only the publish endpoint requires authentication:

- **Publish Endpoint**: Requires a valid GitHub personal access token
- **Header Format**: `Authorization: Bearer YOUR_GITHUB_TOKEN`

## Endpoints

### Health Check

Check the health status of the registry service.

**Endpoint:** `GET /v0/health`

**Response:**
```json
{
  "status": "ok",
  "github_client_configured": true
}
```

---

### List Servers

Retrieve a paginated list of MCP servers with optional filtering, sorting, and search.

**Endpoint:** `GET /v0/servers`

**Query Parameters:**
| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `limit` | integer | Number of results per page (max: 500) | 50 |
| `offset` | integer | Number of results to skip | 0 |
| `registry_name` | string | Filter by package registry (npm, pip, docker, etc.) | - |
| `sort` | string | Sort order (see options below) | newest |
| `search` | string | Search in name and description | - |

**Sort Options:**
- `newest` or `release_date_desc` - Latest releases first (default)
- `release_date_asc` - Oldest releases first
- `alphabetical` or `name_asc` - Alphabetical by name (A-Z)
- `name_desc` - Reverse alphabetical (Z-A)

**Example Request:**
```bash
curl "https://registry.plugged.in/v0/servers?registry_name=npm&sort=newest&limit=10"
```

**Response:**
```json
{
  "servers": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "io.github.username/mcp-server-example",
      "description": "An example MCP server",
      "repository": {
        "url": "https://github.com/username/mcp-server-example",
        "source": "github",
        "id": "123456789"
      },
      "version_detail": {
        "version": "1.2.3",
        "release_date": "2025-01-11T12:00:00Z",
        "is_latest": true
      },
      "packages": [
        {
          "registry_name": "npm",
          "name": "@username/mcp-server-example",
          "version": "1.2.3"
        }
      ]
    }
  ],
  "pagination": {
    "limit": 10,
    "offset": 0,
    "total": 125
  }
}
```

---

### Get Server Details

Retrieve detailed information about a specific MCP server.

**Endpoint:** `GET /v0/servers/{id}`

**Path Parameters:**
- `id` - The UUID of the server

**Example Request:**
```bash
curl "https://registry.plugged.in/v0/servers/550e8400-e29b-41d4-a716-446655440000"
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "io.github.username/mcp-server-example",
  "description": "An example MCP server with extended features",
  "repository": {
    "url": "https://github.com/username/mcp-server-example",
    "source": "github",
    "id": "123456789"
  },
  "version_detail": {
    "version": "1.2.3",
    "release_date": "2025-01-11T12:00:00Z",
    "is_latest": true
  },
  "packages": [
    {
      "registry_name": "npm",
      "name": "@username/mcp-server-example",
      "version": "1.2.3"
    }
  ],
  "capabilities": {
    "tools": ["calculate", "search", "translate"],
    "prompts": ["math_helper", "code_assistant"],
    "resources": ["file_access", "web_fetch"]
  },
  "transports": ["stdio", "http"],
  "license": "MIT",
  "author": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

---

### Publish Server

Publish a new MCP server or update an existing one. Requires GitHub authentication.

**Endpoint:** `POST /v0/publish`

**Headers:**
- `Authorization: Bearer YOUR_GITHUB_TOKEN`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "url": "https://github.com/username/mcp-server-name"
}
```

**Requirements:**
1. Valid GitHub personal access token
2. Token owner must have admin access to the repository
3. Repository must contain a valid MCP server configuration

**Example Request:**
```bash
curl -X POST "https://registry.plugged.in/v0/publish" \
  -H "Authorization: Bearer ghp_xxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/username/mcp-server-example"}'
```

**Success Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Server published successfully"
}
```

**Error Responses:**

401 Unauthorized:
```json
{
  "error": "Invalid or missing authentication token"
}
```

403 Forbidden:
```json
{
  "error": "You don't have permission to publish this repository"
}
```

400 Bad Request:
```json
{
  "error": "Invalid repository URL or missing MCP configuration"
}
```

---

### Refresh Cache

Manually refresh the proxy cache (useful after publishing).

**Endpoint:** `POST /v0/cache/refresh`

**Example Request:**
```bash
curl -X POST "https://registry.plugged.in/v0/cache/refresh"
```

**Response:**
```json
{
  "message": "Cache refreshed successfully",
  "updated_at": "2025-01-11T12:00:00Z"
}
```

---

## Data Models

### Server Object

```typescript
interface Server {
  id: string;                    // UUID
  name: string;                  // Format: "io.github.{owner}/{repo}"
  description: string;           // Short description
  repository: Repository;        // Repository details
  version_detail: VersionDetail; // Version information
  packages: Package[];           // Available packages
}
```

### Repository Object

```typescript
interface Repository {
  url: string;    // Full GitHub URL
  source: string; // "github"
  id: string;     // GitHub repository ID
}
```

### Version Detail Object

```typescript
interface VersionDetail {
  version: string;       // Semantic version (e.g., "1.2.3")
  release_date: string;  // ISO 8601 timestamp
  is_latest: boolean;    // Whether this is the latest version
}
```

### Package Object

```typescript
interface Package {
  registry_name: string; // "npm", "pip", "docker", etc.
  name: string;          // Package name in the registry
  version: string;       // Package version
}
```

### Pagination Object

```typescript
interface Pagination {
  limit: number;  // Results per page
  offset: number; // Number of skipped results
  total: number;  // Total number of results
}
```

---

## Example Use Cases

### 1. Search for Python MCP Servers

```bash
curl "https://registry.plugged.in/v0/servers?registry_name=pip&search=llm"
```

### 2. Get Latest NPM Servers

```bash
curl "https://registry.plugged.in/v0/servers?registry_name=npm&sort=newest&limit=20"
```

### 3. Paginate Through All Servers

```bash
# First page
curl "https://registry.plugged.in/v0/servers?limit=50&offset=0"

# Second page
curl "https://registry.plugged.in/v0/servers?limit=50&offset=50"
```

### 4. Search and Sort

```bash
curl "https://registry.plugged.in/v0/servers?search=assistant&sort=alphabetical"
```

---

## Rate Limiting

Currently, there are no rate limits on the API. However, please be respectful of the service:

- Cache responses when possible
- Use appropriate pagination limits
- Avoid making excessive requests

---

## CORS Configuration

The API supports CORS for the following origins:
- `https://plugged.in`
- `https://staging.plugged.in`
- `http://localhost:12005`

---

## Error Handling

All errors follow a consistent format:

```json
{
  "error": "Error description",
  "code": "ERROR_CODE",
  "details": {
    // Additional error context
  }
}
```

Common HTTP status codes:
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (missing/invalid auth)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `500` - Internal Server Error

---

## SDK Support

While there's no official SDK yet, the API is RESTful and can be easily integrated with any HTTP client library.

### JavaScript/TypeScript Example

```javascript
// List servers
const response = await fetch('https://registry.plugged.in/v0/servers?registry_name=npm');
const data = await response.json();

// Publish server
const publishResponse = await fetch('https://registry.plugged.in/v0/publish', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${githubToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    url: 'https://github.com/username/mcp-server'
  })
});
```

### Python Example

```python
import requests

# List servers
response = requests.get('https://registry.plugged.in/v0/servers', 
                       params={'registry_name': 'pip'})
servers = response.json()

# Publish server
headers = {
    'Authorization': f'Bearer {github_token}',
    'Content-Type': 'application/json'
}
payload = {'url': 'https://github.com/username/mcp-server'}
response = requests.post('https://registry.plugged.in/v0/publish', 
                        headers=headers, json=payload)
```

---

## Webhook Events (Coming Soon)

Future versions will support webhooks for:
- New server published
- Server updated
- Server removed

---

## Support

For issues, questions, or feature requests:
- GitHub Issues: [https://github.com/VeriTeknik/registry-proxy/issues](https://github.com/VeriTeknik/registry-proxy/issues)
- Email: support@plugged.in

---

Last Updated: January 2025