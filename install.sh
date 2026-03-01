#!/bin/bash
set -e

# ─────────────────────────────────────────
#  Aether Installer
#  https://aether.network
# ─────────────────────────────────────────

BOOTSTRAP_URL="${AETHER_BOOTSTRAP:-http://localhost:7070}"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="aether"
REPO_DIR="/opt/aether"
GO_VERSION="1.23.4"
GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"

BOLD="\033[1m"
GREEN="\033[0;32m"
CYAN="\033[0;36m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
RESET="\033[0m"

banner() {
  echo ""
  echo -e "${CYAN}${BOLD}"
  echo "   █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗ "
  echo "  ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗"
  echo "  ███████║█████╗     ██║   ███████║█████╗  ██████╔╝"
  echo "  ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗"
  echo "  ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║"
  echo "  ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝"
  echo -e "${RESET}"
  echo -e "  ${BOLD}P2P Internet — Free, Open, Censorship-Resistant${RESET}"
  echo ""
}

info()    { echo -e "  ${CYAN}→${RESET} $1"; }
success() { echo -e "  ${GREEN}✓${RESET} $1"; }
warn()    { echo -e "  ${YELLOW}!${RESET} $1"; }
error()   { echo -e "  ${RED}✗${RESET} $1"; exit 1; }
step()    { echo -e "\n${BOLD}$1${RESET}"; }

check_root() {
  if [ "$EUID" -ne 0 ]; then
    error "Please run as root: sudo bash install.sh"
  fi
}

check_os() {
  if [ "$(uname -s)" != "Linux" ]; then
    error "Aether currently supports Linux only. macOS and Windows support coming soon."
  fi
  if [ "$(uname -m)" != "x86_64" ]; then
    error "Aether requires an x86_64 system."
  fi
  success "Linux x86_64 detected"
}

install_deps() {
  step "[1/5] Installing dependencies"
  if command -v apt-get &>/dev/null; then
    apt-get update -qq
    apt-get install -y -qq curl git build-essential
  elif command -v yum &>/dev/null; then
    yum install -y -q curl git gcc
  elif command -v pacman &>/dev/null; then
    pacman -Sy --noconfirm curl git base-devel
  else
    warn "Unknown package manager. Make sure curl and git are installed."
  fi
  success "Dependencies ready"
}

install_go() {
  step "[2/5] Setting up Go"
  if command -v go &>/dev/null; then
    INSTALLED=$(go version | awk '{print $3}' | sed 's/go//')
    success "Go ${INSTALLED} already installed"
    export PATH=$PATH:/usr/local/go/bin
    return
  fi

  info "Downloading Go ${GO_VERSION}..."
  curl -fsSL "$GO_URL" -o /tmp/go.tar.gz
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz

  echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
  export PATH=$PATH:/usr/local/go/bin

  success "Go ${GO_VERSION} installed"
}

build_aether() {
  step "[3/5] Building Aether"
  export PATH=$PATH:/usr/local/go/bin

  # Copy source to install location
  mkdir -p "$REPO_DIR"

  # If we're running from the source directory, copy it
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  if [ -f "$SCRIPT_DIR/node/main.go" ]; then
    info "Building from local source..."
    cp -r "$SCRIPT_DIR/node" "$REPO_DIR/"
    cp -r "$SCRIPT_DIR/bootstrap" "$REPO_DIR/"
  else
    error "Source not found. Make sure install.sh is in the aether project directory."
  fi

  # Build node
  info "Building node..."
  cd "$REPO_DIR/node"
  go build -ldflags="-s -w" -o "$INSTALL_DIR/aether-node" .

  # Build bootstrap
  info "Building bootstrap server..."
  cd "$REPO_DIR/bootstrap"
  go build -ldflags="-s -w" -o "$INSTALL_DIR/aether-bootstrap" .

  success "Aether binaries installed to $INSTALL_DIR"
}

create_service() {
  step "[4/5] Setting up system service"

  # Create a dedicated user for the service
  if ! id -u aether &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin aether
    success "Created system user: aether"
  fi

  # Write systemd service
  cat > /etc/systemd/system/aether.service <<EOF
[Unit]
Description=Aether P2P Network Node
Documentation=https://aether.network
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=aether
ExecStart=$INSTALL_DIR/aether-node -bootstrap $BOOTSTRAP_URL
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable aether
  systemctl start aether

  sleep 2

  if systemctl is-active --quiet aether; then
    success "Aether service started and enabled on boot"
  else
    warn "Service may not have started. Check: sudo systemctl status aether"
  fi
}

print_success() {
  step "[5/5] Done"
  echo ""
  echo -e "${GREEN}${BOLD}  Aether is running!${RESET}"
  echo ""
  echo -e "  ${BOLD}Dashboard:${RESET}     http://localhost:8080"
  echo -e "  ${BOLD}SOCKS5 Proxy:${RESET}  localhost:1080"
  echo -e "  ${BOLD}Bootstrap:${RESET}     $BOOTSTRAP_URL"
  echo ""
  echo -e "  ${BOLD}To use Aether in your browser:${RESET}"
  echo -e "  Set your SOCKS5 proxy to ${CYAN}localhost:1080${RESET}"
  echo ""
  echo -e "  ${BOLD}Useful commands:${RESET}"
  echo -e "  ${CYAN}sudo systemctl status aether${RESET}   — check node status"
  echo -e "  ${CYAN}sudo systemctl restart aether${RESET}  — restart the node"
  echo -e "  ${CYAN}sudo journalctl -u aether -f${RESET}   — view live logs"
  echo ""
}

# ─── Main ───
banner
check_root
check_os
install_deps
install_go
build_aether
create_service
print_success
