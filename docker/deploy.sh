#!/bin/bash
set -e

echo "=================================="
echo "  RMM Platform - Hetzner Deploy"
echo "=================================="

if [ ! -f .env.production ]; then
    echo "ERROR: .env.production not found!"
    echo "Copy from .env.production and edit"
    exit 1
fi

export $(grep -v '^#' .env.production | xargs)

echo "1/4 Pulling images & building..."
docker compose build --pull

echo "2/4 Stopping old containers..."
docker compose down --remove-orphans

echo "3/4 Starting services..."
docker compose up -d

echo "4/4 Health check..."
sleep 5
for i in $(seq 1 12); do
    status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/health 2>/dev/null || echo "000")
    if [ "$status" = "200" ]; then
        echo "  System is UP (HTTP $status)"
        break
    fi
    echo "  Waiting... ($i/12)"
    sleep 5
done

echo "=================================="
echo "  Frontend : http://$(curl -s ifconfig.me 2>/dev/null || echo 'SERVER_IP')"
echo "  Swagger  : http://$(curl -s ifconfig.me 2>/dev/null || echo 'SERVER_IP')/swagger/"
echo "  API      : http://$(curl -s ifconfig.me 2>/dev/null || echo 'SERVER_IP')/api/v1/"
echo "=================================="
