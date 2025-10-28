# Security Guidelines

## Credentials Management

**CRITICAL**: Never commit credentials to version control!

### Setup Instructions

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Generate a secure password:
   ```bash
   openssl rand -base64 32
   ```

3. Update all instances of `CHANGEME_SECURE_PASSWORD_HERE` in `.env` with your generated password

4. Generate a JWT private key:
   ```bash
   openssl genrsa -out jwt-private.pem 2048
   # Then copy the contents to MCP_REGISTRY_JWT_PRIVATE_KEY in .env
   ```

5. Generate an API key for the proxy ratings endpoints:
   ```bash
   openssl rand -hex 32
   ```
   Update `API_KEY` in `.env` with this value

6. Set up GitHub OAuth:
   - Go to https://github.com/settings/developers
   - Create a new OAuth App
   - Set the callback URL to: `https://your-domain.com/v0/callback`
   - Copy Client ID and Client Secret to `.env`

### Before Pushing to GitHub

Run this checklist:

- [ ] No hardcoded passwords in any files
- [ ] `.env` is in `.gitignore`
- [ ] All services use environment variables
- [ ] `.env.example` has placeholder values only
- [ ] MIGRATION_STATUS.md is in `.gitignore` (contains deployment-specific info)

### Rotating Credentials

If credentials are accidentally committed:

1. **Immediately** change all passwords in production
2. Rotate GitHub OAuth credentials
3. Generate new JWT keys
4. Use `git filter-branch` or BFG Repo-Cleaner to remove from history
5. Force push to remote (if repository is private)
6. Notify team members to pull fresh copy

### Ratings API Security

The ratings and installation tracking endpoints are protected with API key authentication:

- **POST /v0/servers/:id/rate** - Requires `Authorization: Bearer <API_KEY>` header
- **POST /v0/servers/:id/install** - Requires `Authorization: Bearer <API_KEY>` header
- **GET /v0/servers/:id/reviews** - Public (read-only)
- **GET /v0/servers/:id/stats** - Public (read-only)

The API key should be:
- Stored securely in the plugged.in app's environment variables
- Used by the app backend to submit ratings on behalf of authenticated users
- Rotated regularly (at least quarterly)
- Never exposed to the frontend/client-side code

### Rate Limiting

For production deployments:

- Use Traefik rate limiting middleware
- Implement IP-based throttling
- Add CAPTCHA for suspicious patterns
- Monitor for abuse in application logs

### Database Access

- PostgreSQL is not exposed publicly (internal Docker network only)
- Use strong passwords (minimum 32 characters)
- Regularly backup the database
- Rotate credentials quarterly

### Admin Access

- Admin interface should be behind authentication
- Use HTTPS only
- Implement 2FA for admin users
- Audit log all admin actions

## Reporting Security Issues

If you discover a security vulnerability, please email security@plugged.in with:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

Do not create public issues for security vulnerabilities.
