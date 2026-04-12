#!/usr/bin/env bash
# One-time initial setup on a fresh Ubuntu 22.04 VM.
# Assumes the repo is already cloned. Run from the repo root:
#   bash deployment/setup.sh
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# ── 1. Install Docker ────────────────────────────────────────────────────────
echo "--- installing Docker ---"
sudo apt-get update -q
sudo apt-get install -y -q ca-certificates curl gnupg

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update -q
sudo apt-get install -y -q docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Allow ubuntu user to run docker without sudo
sudo usermod -aG docker ubuntu
echo "--- Docker $(docker --version) installed ---"

# ── 2. Create .env file ──────────────────────────────────────────────────────
if [ ! -f "${REPO_DIR}/.env" ]; then
  echo "--- creating .env template ---"
  cat > "${REPO_DIR}/.env" <<'EOF'
SERVER_PORT=8080
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_CREDENTIALS_JSON={"type":"service_account",...}
OPENAQ_API_KEY=your-openaq-api-key
COUNTRIES_API_URL=http://129.241.150.113:8080/v3.1
METEO_API_URL=https://api.open-meteo.com/v1
OPENAQ_API_URL=https://api.openaq.org/v3
NOMINATIM_API_URL=https://nominatim.openstreetmap.org
CURRENCY_API_URL=http://129.241.150.113:9090/currency
CACHE_PURGE_INTERVAL_HOURS=1
EOF
  chmod 600 "${REPO_DIR}/.env"
else
  echo "--- .env already exists, skipping ---"
fi

echo ""
echo "=== Setup complete ==="
echo "Edit ${REPO_DIR}/.env with your real secrets, then run:"
echo "  cd ${REPO_DIR} && docker compose up --build -d"
echo "  docker compose logs -f"
echo ""
echo "To deploy future updates:"
echo "  ${REPO_DIR}/deployment/deploy.sh"
