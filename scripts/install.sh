#!/bin/bash
set -e

# VaultDrift Installation Script
# Usage: curl -fsSL https://vaultdrift.com/install.sh | bash

REPO="vaultdrift/vaultdrift"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/vaultdrift"
SERVICE_USER="vaultdrift"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    case $OS in
        linux|darwin)
            ;;
        *)
            log_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
}

# Download latest release
download_binary() {
    log_info "Detecting platform..."
    detect_platform

    log_info "Downloading VaultDrift for $PLATFORM..."

    # Get latest release URL
    LATEST_URL=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | \
        grep "browser_download_url.*vaultdrift-$PLATFORM" | \
        cut -d '"' -f 4)

    if [ -z "$LATEST_URL" ]; then
        log_error "Could not find release for platform: $PLATFORM"
        exit 1
    fi

    # Download
    TMP_DIR=$(mktemp -d)
    curl -fsSL "$LATEST_URL" -o "$TMP_DIR/vaultdrift"
    chmod +x "$TMP_DIR/vaultdrift"

    echo "$TMP_DIR/vaultdrift"
}

# Install binary
install_binary() {
    BINARY_PATH=$1

    log_info "Installing binary to $INSTALL_DIR..."

    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_PATH" "$INSTALL_DIR/vaultdrift"
    else
        sudo mv "$BINARY_PATH" "$INSTALL_DIR/vaultdrift"
    fi

    log_info "Binary installed successfully!"
}

# Create user and directories
setup_environment() {
    log_info "Setting up environment..."

    # Create user
    if ! id "$SERVICE_USER" &>/dev/null; then
        log_info "Creating user: $SERVICE_USER"
        sudo useradd -r -s /bin/false -d "$DATA_DIR" "$SERVICE_USER"
    fi

    # Create directories
    sudo mkdir -p "$DATA_DIR/storage"
    sudo chown -R "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"

    log_info "Environment setup complete!"
}

# Install systemd service
install_service() {
    if [ "$OS" != "linux" ]; then
        log_warn "Systemd service installation skipped on $OS"
        return
    fi

    if ! command -v systemctl &> /dev/null; then
        log_warn "systemctl not found, skipping service installation"
        return
    fi

    log_info "Installing systemd service..."

    # Download service file
    SERVICE_URL="https://raw.githubusercontent.com/$REPO/main/deploy/systemd/vaultdrift.service"
    curl -fsSL "$SERVICE_URL" | sudo tee /etc/systemd/system/vaultdrift.service > /dev/null

    # Reload and enable
    sudo systemctl daemon-reload
    sudo systemctl enable vaultdrift

    log_info "Service installed! Start with: sudo systemctl start vaultdrift"
}

# Print post-installation instructions
print_instructions() {
    echo ""
    echo "=========================================="
    echo "  VaultDrift Installation Complete!"
    echo "=========================================="
    echo ""
    echo "Binary location: $INSTALL_DIR/vaultdrift"
    echo "Data directory:  $DATA_DIR"
    echo ""
    echo "Next steps:"
    echo ""
    echo "1. Initialize the server:"
    echo "   sudo vaultdrift init --admin-user admin --admin-email admin@example.com"
    echo ""
    echo "2. Start the server:"
    if command -v systemctl &> /dev/null; then
        echo "   sudo systemctl start vaultdrift"
    else
        echo "   vaultdrift serve"
    fi
    echo ""
    echo "3. Access the web UI:"
    echo "   http://localhost:8080"
    echo ""
    echo "For more information, visit: https://github.com/$REPO"
    echo ""
}

# Main
main() {
    log_info "VaultDrift Installer"
    log_info "===================="
    echo ""

    # Check dependencies
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed"
        exit 1
    fi

    # Download
    BINARY=$(download_binary)

    # Install
    install_binary "$BINARY"
    rm -rf "$(dirname "$BINARY")"

    # Setup
    setup_environment
    install_service

    # Instructions
    print_instructions
}

# Run
main
