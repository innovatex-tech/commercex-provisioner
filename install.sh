#!/bin/bash
set -e

# ╔══════════════════════════════════════════════════════════════════╗
# ║     InnovateX Provisioner - Universal Installer                  ║
# ║     Installs all dependencies + provisioner to ~/.innovatex/     ║
# ╚══════════════════════════════════════════════════════════════════╝

INSTALL_DIR="$HOME/.innovatex/bin"
BINARY_NAME="innovatex"
GO_MIN_VERSION="1.21"
DOCKER_MIN_VERSION="20.0"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}     🚀  ${GREEN}InnovateX Provisioner Installer${NC}  🚀              ${BLUE}║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}▶${NC} $1"
}

print_success() {
    echo -e "  ${GREEN}✓${NC} $1"
}

print_skip() {
    echo -e "  ${YELLOW}⊘${NC} $1 ${YELLOW}(skipped)${NC}"
}

print_warning() {
    echo -e "  ${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "  ${RED}✗${NC} $1"
}

# Version comparison: returns 0 if $1 >= $2
version_gte() {
    [ "$(printf '%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

# Detect OS and package manager
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    elif [ -f /etc/debian_version ]; then
        OS="debian"
    elif [ -f /etc/redhat-release ]; then
        OS="rhel"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
    else
        OS="unknown"
    fi
    echo $OS
}

# ─────────────────────────────────────────────────────────────────────
# Check and Install Go
# ─────────────────────────────────────────────────────────────────────
install_go() {
    print_step "Checking Go installation..."
    
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
        if version_gte "$GO_VERSION" "$GO_MIN_VERSION"; then
            print_skip "Go $GO_VERSION already installed (>= $GO_MIN_VERSION)"
            return 0
        else
            print_warning "Go $GO_VERSION found, but $GO_MIN_VERSION+ required. Updating..."
        fi
    else
        print_warning "Go not found. Installing..."
    fi

    OS=$(detect_os)
    case $OS in
        ubuntu|debian|pop)
            # Use official Go installer for latest version
            GO_LATEST=$(curl -s https://go.dev/VERSION?m=text | head -1)
            curl -LO "https://go.dev/dl/${GO_LATEST}.linux-amd64.tar.gz"
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "${GO_LATEST}.linux-amd64.tar.gz"
            rm "${GO_LATEST}.linux-amd64.tar.gz"
            
            # Add to PATH if not already there
            if ! grep -q '/usr/local/go/bin' ~/.profile 2>/dev/null; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
            fi
            export PATH=$PATH:/usr/local/go/bin
            ;;
        fedora|rhel|centos)
            sudo dnf install -y golang
            ;;
        arch|manjaro)
            sudo pacman -S --noconfirm go
            ;;
        macos)
            if command -v brew &> /dev/null; then
                brew install go
            else
                print_error "Homebrew not found. Please install Go manually: https://go.dev/dl/"
                exit 1
            fi
            ;;
        *)
            print_error "Unknown OS. Please install Go manually: https://go.dev/dl/"
            exit 1
            ;;
    esac
    
    print_success "Go installed: $(go version | grep -oP 'go[0-9]+\.[0-9]+\.[0-9]+')"
}

# ─────────────────────────────────────────────────────────────────────
# Check and Install Docker
# ─────────────────────────────────────────────────────────────────────
install_docker() {
    print_step "Checking Docker installation..."
    
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | grep -oP '[0-9]+\.[0-9]+' | head -1)
        if version_gte "$DOCKER_VERSION" "$DOCKER_MIN_VERSION"; then
            print_skip "Docker $DOCKER_VERSION already installed (>= $DOCKER_MIN_VERSION)"
            
            # Check if docker daemon is running
            if ! docker info &> /dev/null; then
                print_warning "Docker daemon not running. Starting..."
                sudo systemctl start docker 2>/dev/null || true
            fi
            return 0
        else
            print_warning "Docker $DOCKER_VERSION found, but $DOCKER_MIN_VERSION+ required. Updating..."
        fi
    else
        print_warning "Docker not found. Installing..."
    fi

    OS=$(detect_os)
    case $OS in
        ubuntu|debian|pop)
            # Official Docker installation
            sudo apt-get update
            sudo apt-get install -y ca-certificates curl gnupg
            sudo install -m 0755 -d /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/$OS/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            sudo chmod a+r /etc/apt/keyrings/docker.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
            sudo apt-get update
            sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        fedora)
            sudo dnf install -y dnf-plugins-core
            sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
            sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        arch|manjaro)
            sudo pacman -S --noconfirm docker docker-compose
            ;;
        macos)
            if command -v brew &> /dev/null; then
                brew install --cask docker
                print_warning "Please open Docker Desktop to complete installation"
            else
                print_error "Please install Docker Desktop: https://docker.com/products/docker-desktop"
                exit 1
            fi
            ;;
        *)
            print_error "Unknown OS. Please install Docker manually: https://docs.docker.com/get-docker/"
            exit 1
            ;;
    esac
    
    # Start and enable Docker
    if [[ "$OS" != "macos" ]]; then
        sudo systemctl start docker
        sudo systemctl enable docker
        
        # Add current user to docker group
        if ! groups | grep -q docker; then
            sudo usermod -aG docker $USER
            print_warning "Added $USER to docker group. Please log out and back in."
        fi
    fi
    
    print_success "Docker installed: $(docker --version | grep -oP '[0-9]+\.[0-9]+\.[0-9]+')"
}

# ─────────────────────────────────────────────────────────────────────
# Check and Install Docker Compose
# ─────────────────────────────────────────────────────────────────────
install_docker_compose() {
    print_step "Checking Docker Compose installation..."
    
    # Check for docker compose (v2 plugin) or docker-compose (v1 standalone)
    if docker compose version &> /dev/null; then
        COMPOSE_VERSION=$(docker compose version | grep -oP '[0-9]+\.[0-9]+' | head -1)
        print_skip "Docker Compose v$COMPOSE_VERSION already installed (plugin)"
        return 0
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_VERSION=$(docker-compose --version | grep -oP '[0-9]+\.[0-9]+' | head -1)
        print_skip "Docker Compose v$COMPOSE_VERSION already installed (standalone)"
        return 0
    else
        print_warning "Docker Compose not found. Installing..."
    fi

    OS=$(detect_os)
    case $OS in
        ubuntu|debian|pop|fedora)
            # Docker Compose plugin should be installed with Docker
            # If not, install standalone
            sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
            sudo chmod +x /usr/local/bin/docker-compose
            ;;
        arch|manjaro)
            sudo pacman -S --noconfirm docker-compose
            ;;
        macos)
            # Comes with Docker Desktop
            print_warning "Docker Compose comes with Docker Desktop"
            ;;
    esac
    
    print_success "Docker Compose installed"
}

# ─────────────────────────────────────────────────────────────────────
# Install Git (if needed)
# ─────────────────────────────────────────────────────────────────────
install_git() {
    print_step "Checking Git installation..."
    
    if command -v git &> /dev/null; then
        GIT_VERSION=$(git --version | grep -oP '[0-9]+\.[0-9]+' | head -1)
        print_skip "Git $GIT_VERSION already installed"
        return 0
    fi
    
    print_warning "Git not found. Installing..."
    
    OS=$(detect_os)
    case $OS in
        ubuntu|debian|pop)
            sudo apt-get update && sudo apt-get install -y git
            ;;
        fedora|rhel|centos)
            sudo dnf install -y git
            ;;
        arch|manjaro)
            sudo pacman -S --noconfirm git
            ;;
        macos)
            xcode-select --install 2>/dev/null || brew install git
            ;;
    esac
    
    print_success "Git installed: $(git --version | grep -oP '[0-9]+\.[0-9]+\.[0-9]+')"
}

# ─────────────────────────────────────────────────────────────────────
# Install InnovateX Provisioner
# ─────────────────────────────────────────────────────────────────────
install_innovatex() {
    print_step "Installing InnovateX Provisioner..."
    
    # Create installation directory
    mkdir -p "$INSTALL_DIR"
    
    # Set GOBIN to our install directory and install
    GOBIN="$INSTALL_DIR" go install github.com/innovatex-tech/commercex-provisioner/cmd/innovatex@latest
    
    print_success "Installed to $INSTALL_DIR/$BINARY_NAME"
}

# ─────────────────────────────────────────────────────────────────────
# Configure PATH
# ─────────────────────────────────────────────────────────────────────
configure_path() {
    print_step "Configuring PATH..."
    
    # Detect shell RC file
    SHELL_RC=""
    SHELL_NAME=$(basename "$SHELL")
    
    case $SHELL_NAME in
        zsh)  SHELL_RC="$HOME/.zshrc" ;;
        bash) SHELL_RC="$HOME/.bashrc" ;;
        fish) SHELL_RC="$HOME/.config/fish/config.fish" ;;
        *)    SHELL_RC="$HOME/.profile" ;;
    esac
    
    # Add to PATH if not already there
    if ! grep -q '\.innovatex/bin' "$SHELL_RC" 2>/dev/null; then
        echo "" >> "$SHELL_RC"
        echo "# InnovateX Provisioner" >> "$SHELL_RC"
        if [ "$SHELL_NAME" = "fish" ]; then
            echo 'set -gx PATH $HOME/.innovatex/bin $PATH' >> "$SHELL_RC"
        else
            echo 'export PATH="$HOME/.innovatex/bin:$PATH"' >> "$SHELL_RC"
        fi
        print_success "Added to PATH in $SHELL_RC"
    else
        print_skip "PATH already configured in $SHELL_RC"
    fi
    
    # Export for current session
    export PATH="$HOME/.innovatex/bin:$PATH"
}

# ─────────────────────────────────────────────────────────────────────
# Create data directories
# ─────────────────────────────────────────────────────────────────────
create_directories() {
    print_step "Creating data directories..."
    
    mkdir -p "$HOME/.innovatex/clients"
    mkdir -p "$HOME/.innovatex/templates"
    
    print_success "Created ~/.innovatex/clients/"
    print_success "Created ~/.innovatex/templates/"
}

# ─────────────────────────────────────────────────────────────────────
# Main Installation Flow
# ─────────────────────────────────────────────────────────────────────
main() {
    print_header
    
    echo -e "${BLUE}Detected OS:${NC} $(detect_os)"
    echo ""
    
    # Install dependencies
    install_git
    echo ""
    install_go
    echo ""
    install_docker
    echo ""
    install_docker_compose
    echo ""
    
    # Install InnovateX
    install_innovatex
    echo ""
    configure_path
    echo ""
    create_directories
    
    # Final message
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}     ✅  ${GREEN}Installation Complete!${NC}                           ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "📍 ${BLUE}Binary:${NC}    $INSTALL_DIR/$BINARY_NAME"
    echo -e "📁 ${BLUE}Data:${NC}      ~/.innovatex/"
    echo ""
    echo -e "${YELLOW}⚠  Please restart your terminal or run:${NC}"
    echo -e "   ${GREEN}source $SHELL_RC${NC}"
    echo ""
    echo -e "🎯 ${BLUE}Get started:${NC}"
    echo -e "   ${GREEN}innovatex create${NC}"
    echo ""
}

# Run main
main "$@"
