#!/usr/bin/env bash

# GitPatrol - Tactical Asset Monitoring
# Proxmox LXC Installation Script
# Standard: Binary-First Installation (Fast & Lightweight)

set -e

# --- Configuration ---
APP="GitPatrol"
INSTALL_DIR="/opt/gitpatrol"
DATA_DIR="/var/lib/gitpatrol"
DB_DIR="$DATA_DIR/db"
BACKEND_PORT=8080
REPO_OWNER="efinityhub"
REPO_NAME="gitpatrol"
GITHUB_REPO="https://github.com/$REPO_OWNER/$REPO_NAME"

# Source tteck helper functions
source /dev/stdin <<< "$({ wget -qLO - https://github.com/community-scripts/ProxmoxVE/raw/main/misc/install.func; } 2>/dev/null)"

# --- Header ---
header_info

# --- Update System ---
msg_info "Updating system"
apt-get update &>/dev/null
apt-get -y upgrade &>/dev/null
msg_ok "System updated"

# --- Architecture Check ---
ARCH=$(uname -m)
case $ARCH in
    x86_64) BIN_ARCH="amd64" ;;
    aarch64) BIN_ARCH="arm64" ;;
    *) msg_error "Unsupported architecture: $ARCH" ;;
esac

# --- Dependency Check ---
msg_info "Installing dependencies"
DEPS="curl git wget jq"
apt-get install -y -qq $DEPS &>/dev/null
msg_ok "Dependencies installed"

# --- Directory Setup ---
msg_info "Setting up directories"
mkdir -p "$INSTALL_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$DB_DIR"
msg_ok "Directories configured"

# --- Binary Installation (Preferred) ---
msg_info "Fetching latest release info"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | jq -r .tag_name)

if [ "$LATEST_RELEASE" != "null" ]; then
    msg_info "Found version $LATEST_RELEASE"
    BINARY_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$LATEST_RELEASE/gitpatrol-linux-$BIN_ARCH"
    
    msg_info "Downloading pre-built binary"
    if wget -q --spider "$BINARY_URL"; then
        wget -q -O "$INSTALL_DIR/gitpatrol" "$BINARY_URL"
        chmod +x "$INSTALL_DIR/gitpatrol"
        msg_ok "Binary installed successfully"
    else
        msg_info "No binary for $BIN_ARCH found. Falling back to source build"
        BUILD_FROM_SOURCE=true
    fi
else
    msg_info "No releases found. Falling back to source build (main branch)"
    BUILD_FROM_SOURCE=true
fi

# --- Source Build Fallback (Only if needed) ---
if [ "$BUILD_FROM_SOURCE" = true ]; then
    msg_info "Starting source build (this will take longer)"
    
    # Install build-only dependencies
    apt-get install -y -qq build-essential &>/dev/null
    
    # Go (if missing)
    if ! command -v go &> /dev/null; then
        msg_info "Installing Go"
        GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | head -n 1)
        wget -q "https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz"
        tar -C /usr/local -xzf "${GO_VERSION}.linux-amd64.tar.gz"
        export PATH=$PATH:/usr/local/go/bin
        rm "${GO_VERSION}.linux-amd64.tar.gz"
        WAS_GO_INSTALLED=true
    fi

    # Node.js (if missing)
    if ! command -v npm &> /dev/null; then
        msg_info "Installing Node.js for build"
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash - &>/dev/null
        apt-get install -y -qq nodejs &>/dev/null
        WAS_NODE_INSTALLED=true
    fi

    # Clone & Build
    msg_info "Cloning source"
    if [ ! -d "$INSTALL_DIR/source" ]; then
        git clone "$GITHUB_REPO" "$INSTALL_DIR/source" &>/dev/null
    else
        cd "$INSTALL_DIR/source" && git pull origin main &>/dev/null
    fi

    msg_info "Building frontend"
    cd "$INSTALL_DIR/source/frontend"
    npm install --silent &>/dev/null && npm run build --silent &>/dev/null
    
    msg_info "Building backend"
    cd "$INSTALL_DIR/source/backend"
    mkdir -p cmd/gitpatrol/build
    cp -r ../frontend/build/* cmd/gitpatrol/build/
    go build -o "$INSTALL_DIR/gitpatrol" ./cmd/gitpatrol/main.go &>/dev/null
    
    # --- Cleanup Build Deps (Surgical) ---
    if [ "$WAS_NODE_INSTALLED" = true ]; then
        msg_info "Cleaning up Node.js"
        apt-get remove -y -qq nodejs &> /dev/null
    fi
    rm -rf "$INSTALL_DIR/source"
    msg_ok "Source build complete"
fi

# --- Environment & DB ---
msg_info "Configuring environment"
if [ ! -f "$DB_DIR/gitpatrol.env" ]; then
    cat > "$DB_DIR/gitpatrol.env" <<EOF
CONFIG_PATH=$DB_DIR/gitpatrol.env
DB_PATH=$DB_DIR/gitpatrol.db
DATA_DIR=$DATA_DIR/data
WORKERS=3
PORT=$BACKEND_PORT
EOF
fi
msg_ok "Environment ready"

# --- Systemd Integration ---
msg_info "Registering Systemd service"
cat > /etc/systemd/system/gitpatrol.service <<EOF
[Unit]
Description=GitPatrol - Tactical Asset Monitoring
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/gitpatrol
Restart=always
RestartSec=5
EnvironmentFile=$DB_DIR/gitpatrol.env

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now gitpatrol.service &>/dev/null
msg_ok "Service started on port $BACKEND_PORT"

# --- Cleanup ---
msg_info "Cleaning up"
apt-get autoremove -y &>/dev/null
apt-get autoclean -y &>/dev/null
msg_ok "System cleaned"
