#!/usr/bin/env bash
# One-time initial setup on a fresh Ubuntu 22.04 VM.
# Run as the ubuntu user: bash setup.sh
set -euo pipefail

REPO_URL="https://github.com/SigurdRiseth/Air-Quality-Environment-Dashboard-Service.git"

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

# ── 2. Clone repo ────────────────────────────────────────────────────────────
echo "--- cloning repository ---"
git clone "${REPO_URL}" /home/ubuntu/envdash
cd /home/ubuntu/envdash

# ── 3. Create .env file ──────────────────────────────────────────────────────
echo "--- creating .env (edit this with your secrets!) ---"
cat > /home/ubuntu/envdash/.env <<'EOF'
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
chmod 600 /home/ubuntu/envdash/.env

echo ""
echo "=== Setup complete ==="
echo "Edit /home/ubuntu/envdash/.env with your real secrets, then run:"
echo "  cd /home/ubuntu/envdash && docker compose up -d"
echo "  docker compose logs -f"
echo ""
echo "To deploy future updates:"
echo "  cd /home/ubuntu/envdash && ./deployment/deploy.sh"
