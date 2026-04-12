#!/usr/bin/env bash
# Pull latest changes and redeploy the service.
# Usage: ./deployment/deploy.sh
set -euo pipefail

cd "$(dirname "$0")/.."

echo "--- pulling latest changes ---"
git pull

echo "--- rebuilding and restarting container ---"
docker compose up --build -d

echo "--- service status ---"
docker compose ps

echo "--- done ---"
