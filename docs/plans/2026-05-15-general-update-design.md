# General Infrastructure Update

**Date:** 2026-05-15
**Status:** Approved

## Problem

Traefik v2.11 Docker client API (1.24) is incompatible with the current Docker Engine (API 1.52, minimum 1.44). This causes complete routing failure — all external requests to `registry.plugged.in` return 404. Additionally, the MCP registry submodule is 108 commits behind upstream.

## Changes

### 1. Traefik v2.11 -> v3.4 (Critical)
- Update image in `main/docker-compose.yml` and `main/docker-compose.traefik.yml`
- Remove deprecated `version: '3.8'` from compose files
- Docker label syntax is backward compatible, no label changes needed

### 2. MCP Registry Submodule Sync
- Update `registry/` submodule from current (v1.1.0+236) to upstream/main (v1.7.9)
- Key upstream changes: SSRF fix, open redirect fix, XSS hardening, OCI rate limiting, OIDC improvements

### 3. PostgreSQL Image Pull
- Keep `postgres:16-alpine`, pull latest patch via image rebuild

### 4. Proxy Cleanup
- Rebuild proxy with current code

## Risks
- Traefik v3 migration is backward compatible for simple Host() rules (confirmed)
- Submodule update may require registry service rebuild
- Brief downtime during Traefik restart (seconds)
