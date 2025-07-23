#!/bin/bash

set -e

# Configuration - Update these variables for your app
REPO="kcterala/kcx"            # GitHub repository (owner/repo)
APP_NAME="kcx"                      # Binary name

# Try different install directories in order of preference
INSTALL_DIRS=("$HOME/.local/bin" "$HOME/bin" "$HOME/.bin" "$HOME/tools" "/usr/local/bin")
INSTALL_DIR=""  # Will be determined by find_install_dir()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
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
    local os arch
    
    case "$(uname -s)" in
        Darwin*)    os="Darwin" ;;
        Linux*)     os="Linux" ;;
        CYGWIN*|MINGW*|MSYS*) os="Windows" ;;
        *)          log_error "Unsupported operating system: $(uname -s)"; exit 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        arm64|aarch64)  arch="arm64" ;;
        armv7l)         arch="arm" ;;
        i386|i686)      arch="i386" ;;
        *)              log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
    
    echo "${os}_${arch}"
}

# Find the best installation directory
find_install_dir() {
    # If user specified INSTALL_DIR environment variable, use it
    if [ -n "${INSTALL_DIR:-}" ]; then
        if [ -d "$INSTALL_DIR" ] || mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            echo "$INSTALL_DIR"
            return 0
        else
            log_warn "Cannot create specified INSTALL_DIR: $INSTALL_DIR"
        fi
    fi
    
    # Try directories in order of preference
    for dir in "${INSTALL_DIRS[@]}"; do
        # Skip if directory is empty (in case of unset HOME)
        [ -n "$dir" ] || continue
        
        if [ -w "$dir" ] || { [ ! -d "$dir" ] && mkdir -p "$dir" 2>/dev/null; }; then
            echo "$dir"
            return 0
        fi
    done
    
    log_error "Could not find a suitable installation directory"
    log_error "Tried: ${INSTALL_DIRS[*]}"
    log_error "Please ensure one of these directories exists and is writable, or set INSTALL_DIR environment variable"
    exit 1
}

# Check if directory is in PATH and warn if not
check_path_warning() {
    local install_dir="$1"
    
    # Check if the directory is in PATH
    if [[ ":$PATH:" != *":$install_dir:"* ]]; then
        log_warn "$install_dir is not in your PATH"
        
        # Provide helpful suggestions based on the directory
        case "$install_dir" in
            "$HOME/.local/bin")
                log_warn "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
                log_warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
                ;;
            "$HOME/bin")
                log_warn "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
                log_warn "  export PATH=\"\$HOME/bin:\$PATH\""
                ;;
            "$HOME/.bin")
                log_warn "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
                log_warn "  export PATH=\"\$HOME/.bin:\$PATH\""
                ;;
            "$HOME/tools")
                log_warn "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
                log_warn "  export PATH=\"\$HOME/tools:\$PATH\""
                ;;
        esac
        log_warn "Then restart your shell or run: source ~/.bashrc (or ~/.zshrc)"
    fi
}
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        log_error "Failed to get latest version from GitHub API"
        exit 1
    fi
    
    echo "$version"
}

# Download and install the binary
install_binary() {
    local version="$1"
    local platform="$2"
    local tmp_dir="/tmp/${APP_NAME}_install"
    local file_extension="tar.gz"
    local extract_cmd="tar -xzf"
    
    # Windows uses zip files
    if [[ "$platform" == Windows_* ]]; then
        file_extension="zip"
        extract_cmd="unzip -q"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/${APP_NAME}_${platform}.${file_extension}"
    
    log_info "Creating temporary directory: $tmp_dir"
    mkdir -p "$tmp_dir"
    
    log_info "Downloading ${APP_NAME} ${version} for ${platform}..."
    if ! curl -L -o "${tmp_dir}/${APP_NAME}.${file_extension}" "$download_url"; then
        log_error "Failed to download from: $download_url"
        log_error "Please check if the release exists and the URL format is correct"
        exit 1
    fi
    
    log_info "Extracting archive..."
    cd "$tmp_dir"
    
    # Check if unzip is available for Windows archives
    if [[ "$file_extension" == "zip" ]] && ! command -v unzip >/dev/null 2>&1; then
        log_error "unzip command not found, required for Windows archives"
        exit 1
    fi
    
    $extract_cmd "${APP_NAME}.${file_extension}"
    
    # Find the binary (it might be in a subdirectory)
    local binary_path
    local binary_name="$APP_NAME"
    
    # On Windows, the binary might have .exe extension
    if [[ "$platform" == Windows_* ]]; then
        binary_name="${APP_NAME}.exe"
    fi
    
    if [ -f "$binary_name" ]; then
        binary_path="$binary_name"
    elif [ -f "./$binary_name" ]; then
        binary_path="./$binary_name"
    else
        # Search for the binary in subdirectories
        binary_path=$(find . -name "$binary_name" -type f | head -n 1)
        if [ -z "$binary_path" ]; then
            log_error "Could not find $binary_name binary in the archive"
            exit 1
        fi
    fi
    
    log_info "Installing ${APP_NAME} to ${INSTALL_DIR}..."
    
    # The install directory should already be writable (selected by find_install_dir)
    # But let's double-check and use sudo as fallback for system directories
    if [ -w "$INSTALL_DIR" ]; then
        cp "$binary_path" "${INSTALL_DIR}/${APP_NAME}"
        chmod +x "${INSTALL_DIR}/${APP_NAME}"
    else
        if command -v sudo >/dev/null 2>&1; then
            log_info "Using sudo for installation to ${INSTALL_DIR}..."
            sudo cp "$binary_path" "${INSTALL_DIR}/${APP_NAME}"
            sudo chmod +x "${INSTALL_DIR}/${APP_NAME}"
        else
            log_error "No write permission to ${INSTALL_DIR} and sudo not available"
            exit 1
        fi
    fi
    
    # Cleanup
    log_info "Cleaning up temporary files..."
    rm -rf "$tmp_dir"
    
    log_info "Installation completed successfully!"
}

# Verify installation
verify_installation() {
    if command -v "${APP_NAME}" >/dev/null 2>&1; then
        local installed_version
        installed_version=$("${APP_NAME}" --version 2>/dev/null || "${APP_NAME}" version 2>/dev/null || echo "unknown")
        log_info "${APP_NAME} is now available in your PATH"
        log_info "Installed version: ${installed_version}"
    else
        log_info "${APP_NAME} was installed to ${INSTALL_DIR}"
        check_path_warning "$INSTALL_DIR"
    fi
}

# Main installation process
main() {
    log_info "Starting installation of ${APP_NAME}..."
    
    # Check dependencies
    local required_cmds="curl"
    
    # Add tar or unzip based on platform
    local platform
    platform=$(detect_platform)
    
    if [[ "$platform" == Windows_* ]]; then
        required_cmds="$required_cmds unzip"
    else
        required_cmds="$required_cmds tar"
    fi
    
    for cmd in $required_cmds; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            log_error "Required command not found: $cmd"
            exit 1
        fi
    done
    
    # Determine installation directory
    INSTALL_DIR=$(find_install_dir)
    log_info "Selected installation directory: $INSTALL_DIR"
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"
    
    # Get latest version
    local version
    version=$(get_latest_version)
    log_info "Latest version: $version"
    
    # Install binary
    install_binary "$version" "$platform"
    
    # Verify installation
    verify_installation
    
    log_info "🎉 ${APP_NAME} has been successfully installed!"
    log_info "Run '${APP_NAME} --help' to get started"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --version, -v  Install specific version"
        echo ""
        echo "Environment variables:"
        echo "  INSTALL_DIR    Override installation directory"
        echo "                 (tries: ~/.local/bin, ~/bin, ~/.bin, ~/tools, /usr/local/bin)"
        echo ""
        echo "Example:"
        echo "  curl -sSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash"
        exit 0
        ;;
    --version|-v)
        if [ -z "${2:-}" ]; then
            log_error "Version argument required"
            exit 1
        fi
        VERSION="$2"
        ;;
esac

# Override install directory if specified
# (This is handled in find_install_dir() function now)

# Run main installation
main