#!/bin/bash

echo "Starting plugged.in infrastructure..."

# Ensure traefik network exists
docker network create traefik 2>/dev/null || true

# Start infrastructure services (Traefik + PostgreSQL)
echo "Starting infrastructure services..."
docker compose up -d

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
sleep 5

# Start Registry (internal)
echo "Starting MCP Registry (internal)..."
cd ../registry
docker compose -f docker-compose.prod.yml up -d

# Start Registry Proxy (public)
echo "Starting Registry Proxy (public interface)..."
cd ../proxy
docker compose -f docker-compose-replace.yml up -d

cd ../main

echo "Waiting for services to be ready..."
sleep 10

echo ""
echo "Services started!"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "Services available at:"
echo "- Registry API: https://registry.plugged.in (enhanced with filtering/sorting)"
echo "- Traefik Dashboard: http://localhost:8080"

echo ""
echo "Checking service health..."
echo -n "- Registry Core: "
if docker exec registry wget -qO- http://localhost:8080/v0/health | grep -q "ok"; then
    echo "✅ OK"
else
    echo "❌ Failed"
fi

echo -n "- Registry Proxy: "
if curl -sf https://registry.plugged.in/v0/health | grep -q "ok"; then
    echo "✅ OK"
else
    echo "❌ Failed"
fi

echo ""
echo "To view logs:"
echo "  docker logs -f registry       # Core registry"
echo "  docker logs -f registry-proxy # Proxy service"
echo "  docker logs -f postgresql     # Database"