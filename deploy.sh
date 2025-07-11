#!/bin/bash
# Deployment script for registry.plugged.in
# This script is called by GitHub Actions to deploy updates

set -e

echo "Starting deployment of registry.plugged.in..."

# Configuration
DEPLOY_DIR="/home/pluggedin/registry/registry"
DOCKER_COMPOSE_FILE="docker-compose.prod.yml"

# Function to check if command was successful
check_status() {
    if [ $? -eq 0 ]; then
        echo "✓ $1 successful"
    else
        echo "✗ $1 failed"
        exit 1
    fi
}

# Navigate to deployment directory
cd "$DEPLOY_DIR"
check_status "Directory navigation"

# Pull latest changes
echo "Pulling latest changes from repository..."
git fetch origin main
git reset --hard origin/main
check_status "Git pull"

# Build new Docker image
echo "Building new Docker image..."
docker build -t registry:upstream .
check_status "Docker build"

# Stop current container
echo "Stopping current registry container..."
docker compose -f "$DOCKER_COMPOSE_FILE" down
check_status "Container stop"

# Start new container
echo "Starting new registry container..."
docker compose -f "$DOCKER_COMPOSE_FILE" up -d
check_status "Container start"

# Wait for service to be ready
echo "Waiting for service to be ready..."
sleep 5

# Health check
echo "Performing health check..."
HEALTH_CHECK=$(curl -s -o /dev/null -w "%{http_code}" https://registry.plugged.in/v0/health)
if [ "$HEALTH_CHECK" = "200" ]; then
    echo "✓ Health check passed"
    echo "✓ Deployment completed successfully!"
else
    echo "✗ Health check failed (HTTP $HEALTH_CHECK)"
    echo "Rolling back..."
    docker compose -f "$DOCKER_COMPOSE_FILE" down
    docker tag registry:upstream-backup registry:upstream 2>/dev/null || true
    docker compose -f "$DOCKER_COMPOSE_FILE" up -d
    exit 1
fi

# Backup current image for rollback
docker tag registry:upstream registry:upstream-backup

echo "Deployment completed at $(date)"