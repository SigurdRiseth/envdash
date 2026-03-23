#!/usr/bin/env bash
# Run this on the VM to pull latest changes and restart the service.
# Usage: ./deployment/deploy.sh
set -euo pipefail

cd /home/ubuntu/envdash

echo "--- pulling latest changes ---"
git pull

echo "--- building binary ---"
go build -o bin/envdash ./cmd/server

echo "--- restarting service ---"
sudo systemctl restart envdash
sudo systemctl status envdash --no-pager

echo "--- done ---"
