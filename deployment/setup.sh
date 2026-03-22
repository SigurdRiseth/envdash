#!/usr/bin/env bash
# One-time initial setup on a fresh Ubuntu 22.04 VM.
# Run as the ubuntu user: bash setup.sh
set -euo pipefail

GO_VERSION="1.26.1"
REPO_URL="https://github.com/YOUR_USERNAME/envdash.git"  # replace with your repo URL

# ── 1. Install Go ────────────────────────────────────────────────────────────
echo "--- installing Go ${GO_VERSION} ---"
wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
rm "go${GO_VERSION}.linux-amd64.tar.gz"

# Add Go to PATH for this session and future logins
export PATH=$PATH:/usr/local/go/bin
grep -qxF 'export PATH=$PATH:/usr/local/go/bin' ~/.profile \
  || echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
go version

# ── 2. Clone repo ────────────────────────────────────────────────────────────
echo "--- cloning repository ---"
git clone "${REPO_URL}" /home/ubuntu/envdash
mkdir -p /home/ubuntu/envdash/bin

# ── 3. Build binary ──────────────────────────────────────────────────────────
echo "--- building binary ---"
cd /home/ubuntu/envdash
go build -o bin/envdash ./cmd/server

# ── 4. Create env file ───────────────────────────────────────────────────────
echo "--- creating /etc/envdash/env (edit this with your secrets!) ---"
sudo mkdir -p /etc/envdash
sudo tee /etc/envdash/env > /dev/null <<'EOF'
SERVER_PORT=8080
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_CREDENTIALS_JSON={"type":"service_account",...}
OPENAQ_API_KEY=your-openaq-api-key
COUNTRIES_API_URL=http://129.241.150.113:8080/v3.1
METEO_API_URL=https://api.open-meteo.com/v1
OPENAQ_API_URL=https://api.openaq.org/v3
NOMINATIM_API_URL=https://nominatim.openstreetmap.org
CURRENCY_API_URL=http://129.241.150.113:9090/currency
EOF
sudo chmod 600 /etc/envdash/env

# ── 5. Install systemd service ───────────────────────────────────────────────
echo "--- installing systemd service ---"
sudo cp /home/ubuntu/envdash/deployment/envdash.service /etc/systemd/system/envdash.service
sudo systemctl daemon-reload
sudo systemctl enable envdash

echo ""
echo "=== Setup complete ==="
echo "Edit /etc/envdash/env with your real secrets, then run:"
echo "  sudo systemctl start envdash"
echo "  sudo systemctl status envdash"
echo ""
echo "To deploy future updates from your local machine:"
echo "  ssh ubuntu@<FLOATING_IP> 'cd envdash && ./deployment/deploy.sh'"
