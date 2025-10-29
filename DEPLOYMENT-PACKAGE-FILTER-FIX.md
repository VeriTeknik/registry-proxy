# Deployment Guide: Package Type Filter Fix

## Summary
Fixed the package registry type filtering in the PostgreSQL query. Changed field name from `registryType` to `registry_name` to match the actual JSON structure.

## Issue
When users select package types (npm, pypi, remote, etc.) on the `/search` page, 0 servers were returned because the database query was looking for the wrong JSON field name.

## Fix Applied
**File**: `proxy/internal/db/postgres.go`
**Line**: 414
**Change**: `p->>'registryType'` → `p->>'registry_name'`

## Impact
- ✅ Fixes npm package filter
- ✅ Fixes pypi package filter
- ✅ Fixes oci/docker package filter
- ✅ Fixes remote (SSE/HTTP) server filter
- ✅ Fixes combined filters (e.g., npm+remote)

## Deployment Steps

### Prerequisites
- SSH access to production server
- Docker installed on server
- Access to registry-proxy repository

### Step 1: Backup Current Deployment
```bash
# SSH to production server
ssh user@registry.plugged.in

# Create backup of current image
docker tag registry-proxy:latest registry-proxy:backup-$(date +%Y%m%d)

# Backup database (recommended)
docker exec mongodb mongodump --out=/backup/$(date +%Y%m%d)
```

### Step 2: Deploy Fixed Image

#### Option A: Build on Server (Recommended)
```bash
# SSH to production server
ssh user@registry.plugged.in

# Navigate to registry-proxy directory
cd /path/to/registry-proxy

# Pull latest changes
git fetch origin
git checkout fix/package-filter-field-name
git pull

# Build new image
cd proxy
docker build -t registry-proxy:latest .

# Restart service
cd ../main
docker-compose down registry-proxy
docker-compose up -d registry-proxy

# Verify deployment
docker logs -f registry-proxy
```

#### Option B: Push from Local and Pull on Server
```bash
# On local machine, push to Docker registry
docker tag registry-proxy:latest your-registry/registry-proxy:v2.1
docker push your-registry/registry-proxy:v2.1

# On production server
ssh user@registry.plugged.in
docker pull your-registry/registry-proxy:v2.1
docker tag your-registry/registry-proxy:v2.1 registry-proxy:latest

# Restart service
cd /path/to/registry-proxy/main
docker-compose down registry-proxy
docker-compose up -d registry-proxy
```

### Step 3: Verify Deployment

#### Test Package Type Filters
```bash
# Test npm filter
curl "https://registry.plugged.in/v0/enhanced/servers?registry_types=npm&limit=5"

# Test pypi filter
curl "https://registry.plugged.in/v0/enhanced/servers?registry_types=pypi&limit=5"

# Test remote filter
curl "https://registry.plugged.in/v0/enhanced/servers?registry_types=remote&limit=5"

# Test combined filters
curl "https://registry.plugged.in/v0/enhanced/servers?registry_types=npm,remote&limit=5"

# Each should return results (not empty array)
```

#### Check Logs
```bash
# Monitor for errors
docker logs --tail=100 -f registry-proxy

# Should see successful requests with no errors
```

#### Test in Frontend
1. Visit `https://app.plugged.in/search`
2. Select "Remote" package type → Should show remote servers
3. Select "npm" package type → Should show npm servers
4. Select multiple types → Should show combined results
5. Verify pagination works correctly

### Step 4: Monitor

#### Check Metrics
```bash
# Check container health
docker ps | grep registry-proxy

# Check memory/CPU usage
docker stats registry-proxy --no-stream

# Check recent errors
docker logs registry-proxy | grep -i error | tail -20
```

#### Verify API Performance
```bash
# Response time test
time curl "https://registry.plugged.in/v0/enhanced/servers?registry_types=npm&limit=100"

# Should complete in < 2 seconds
```

## Rollback Plan

If issues occur, rollback to previous version:

```bash
# Stop current service
docker-compose down registry-proxy

# Restore backup image
docker tag registry-proxy:backup-YYYYMMDD registry-proxy:latest

# Restart service
docker-compose up -d registry-proxy

# Verify rollback
curl "https://registry.plugged.in/v0/health"
```

## Testing Checklist

- [ ] npm filter returns results
- [ ] pypi filter returns results
- [ ] remote filter returns results
- [ ] oci/docker filter returns results
- [ ] Combined filters work (npm+remote)
- [ ] Pagination works with filters
- [ ] No errors in logs
- [ ] Response times acceptable (< 2s)
- [ ] Frontend /search page works
- [ ] All package type chips functional

## Post-Deployment

### Merge to Main
Once verified working in production:

```bash
git checkout main
git merge fix/package-filter-field-name
git push origin main
```

### Update Documentation
Document this fix in:
- CHANGELOG.md
- registry-proxy README
- API documentation

### Notify Users
If this was a critical bug affecting users:
- Post in announcements channel
- Update status page
- Send email to affected users

## Technical Notes

### Why This Bug Occurred
The PostgreSQL query was using `registryType` (camelCase) but the actual JSON field is `registry_name` (snake_case). This is the same type of field name inconsistency we fixed in the pagination implementation.

### Related Files
- `/proxy/internal/db/postgres.go` - Database query
- `/proxy/internal/handlers/enhanced.go` - API handler
- Frontend: `/app/api/service/search/route.ts` - Calls registry-proxy

### Database Structure
```json
{
  "packages": [
    {
      "registry_name": "npm",  // ← Correct field name
      "name": "package-name",
      "version": "1.0.0"
    }
  ]
}
```

## Support

If issues occur during deployment:
- Check logs: `docker logs registry-proxy`
- Verify database connectivity: `docker exec registry-proxy ping mongodb`
- Test endpoint directly: `curl localhost:8090/v0/enhanced/servers`
- Contact: [support contact]

## Changelog

- **2025-01-29**: Fixed package type filtering by correcting field name from `registryType` to `registry_name`
- **Impact**: All package type filters now work correctly
- **Branch**: `fix/package-filter-field-name`
- **Commits**: See git log for details
