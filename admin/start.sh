#!/bin/bash

echo "Starting Registry Admin Service..."

# Build the latest image
echo "Building admin image..."
docker build -t registry-admin:latest .

# Start the service
echo "Starting admin service with Traefik..."
docker compose -f docker-compose.prod.yml up -d

echo "Admin service started!"
echo "Access at: https://admin.registry.plugged.in/"
echo ""
echo "Note: Admin credentials are configured in .env file"