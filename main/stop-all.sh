#!/bin/bash

echo "Stopping all plugged.in services..."

# Stop proxy service
echo "Stopping Registry Proxy..."
cd ../proxy
docker compose -f docker-compose-replace.yml down 2>/dev/null || true

# Stop registry services
echo "Stopping Registry Core..."
cd ../registry
docker compose -f docker-compose.prod.yml down 2>/dev/null || true
docker compose -f docker-compose.local.yml down 2>/dev/null || true

cd ../main

# Stop infrastructure services
echo "Stopping infrastructure services..."
docker compose down

echo ""
echo "All services stopped."