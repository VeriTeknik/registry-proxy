# Deployment Setup for registry.plugged.in

This document describes the CI/CD setup for automatic deployment of the MCP Registry to https://registry.plugged.in.

## Overview

The deployment pipeline consists of:
1. **Testing** - Runs Go tests and linting on every push/PR
2. **Building** - Builds Docker image on main branch pushes
3. **Deployment** - Deploys to production server automatically

## GitHub Actions Workflows

### 1. Simple Deploy (`deploy.yml`)
- Triggers on pushes to main branch
- Runs deployment script via SSH
- Verifies deployment health

### 2. Full CI/CD Pipeline (`ci-cd.yml`)
- Runs tests and linting
- Builds and tests Docker image
- Deploys only after successful tests
- Creates deployment records

## Required GitHub Secrets

Configure these in your repository settings under **Settings → Secrets and variables → Actions**:

### Deployment Secrets

| Secret Name | Description | Example Value |
|------------|-------------|---------------|
| `DEPLOY_HOST` | Server hostname or IP | `registry.plugged.in` or `192.168.1.100` |
| `DEPLOY_USER` | SSH username | `pluggedin` |
| `DEPLOY_PORT` | SSH port (optional) | `22` |
| `DEPLOY_SSH_KEY` | Private SSH key for deployment | See below for generation |

### Generating SSH Key for Deployment

1. On your local machine, generate a new SSH key pair:
   ```bash
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f deploy_key
   ```

2. Add the public key to the server:
   ```bash
   ssh-copy-id -i deploy_key.pub pluggedin@your-server
   # Or manually add to ~/.ssh/authorized_keys on the server
   ```

3. Copy the private key content:
   ```bash
   cat deploy_key
   ```

4. Add the private key to GitHub Secrets as `DEPLOY_SSH_KEY`

## Deployment Script

The deployment script (`/home/pluggedin/registry/deploy.sh`) performs:

1. **Git Pull** - Fetches latest changes from repository
2. **Docker Build** - Builds new registry image
3. **Container Replacement** - Stops old, starts new container
4. **Health Check** - Verifies service is running
5. **Rollback** - Automatically rolls back on failure

## Manual Deployment

To manually trigger deployment:
1. Go to **Actions** tab in GitHub
2. Select "Deploy to Production" or "CI/CD Pipeline"
3. Click "Run workflow"
4. Select branch and run

## Monitoring Deployments

- Check GitHub Actions tab for deployment status
- View logs at: `docker logs registry`
- Monitor health at: https://registry.plugged.in/v0/health

## Environment Configuration

The registry uses environment variables from `.env` file:
- `MCP_REGISTRY_DATABASE_URL`
- `MCP_REGISTRY_GITHUB_CLIENT_ID`
- `MCP_REGISTRY_GITHUB_CLIENT_SECRET`
- `MCP_REGISTRY_ENVIRONMENT`

## Rollback Procedure

If deployment fails:
1. The script automatically attempts rollback
2. Manual rollback:
   ```bash
   cd /home/pluggedin/registry/registry
   docker compose -f docker-compose.prod.yml down
   docker tag registry:upstream-backup registry:upstream
   docker compose -f docker-compose.prod.yml up -d
   ```

## Security Notes

- SSH key should have minimal permissions
- Consider using deployment-specific user
- Regularly rotate SSH keys
- Monitor access logs

## Troubleshooting

### Deployment Fails
- Check GitHub Actions logs
- Verify SSH connectivity
- Check server disk space
- Verify Docker daemon is running

### Health Check Fails
- Check container logs: `docker logs registry`
- Verify MongoDB is running
- Check network connectivity
- Verify Traefik configuration

## Future Improvements

- [ ] Add staging environment
- [ ] Implement blue-green deployments
- [ ] Add automated backups before deployment
- [ ] Set up deployment notifications (Slack/Discord)
- [ ] Add performance testing in pipeline