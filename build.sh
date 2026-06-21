#!/bin/bash
# ==============================================================================
# NEXUS RESEARCH STATION - BUILD SCRIPT
# ==============================================================================
# This script handles building the Nexus Research Station application.
# It checks dependencies, builds the frontend assets, and compiles the Go backend.
# ==============================================================================

set -e

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Ensure we run from the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Help output
show_help() {
    echo "Usage: ./build.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -c, --clean         Clean previous build artifacts and dependencies before building"
    echo "  -f, --frontend      Build only the frontend assets"
    echo "  -b, --backend       Build only the Go backend (requires existing frontend assets)"
    echo "  -t, --test          Run backend unit tests with coverage after building"
    echo "  -h, --help          Show this help message and exit"
    echo ""
}

# Parse options
CLEAN_BUILD=false
BUILD_FRONTEND=true
BUILD_BACKEND=true
RUN_TESTS=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -c|--clean)
            CLEAN_BUILD=true
            shift
            ;;
        -f|--frontend)
            BUILD_BACKEND=false
            shift
            ;;
        -b|--backend)
            BUILD_FRONTEND=false
            shift
            ;;
        -t|--test)
            RUN_TESTS=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option '$1'${NC}"
            show_help
            exit 1
            ;;
    esac
done

echo -e "${BLUE}=============================================${NC}"
echo -e "${BLUE}  NEXUS RESEARCH STATION - BUILD SYSTEM      ${NC}"
echo -e "${BLUE}=============================================${NC}"

# Check for Go if compiling backend
if [ "$BUILD_BACKEND" = true ]; then
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go compiler is not installed or not in PATH.${NC}"
        exit 1
    fi
fi

# Check for Node/NPM if compiling frontend
if [ "$BUILD_FRONTEND" = true ]; then
    if ! command -v npm &> /dev/null; then
        echo -e "${RED}Error: npm is not installed or not in PATH.${NC}"
        exit 1
    fi
fi

# 1. Clean Build if requested
if [ "$CLEAN_BUILD" = true ]; then
    echo -e "${YELLOW}Cleaning build artifacts and dependencies...${NC}"
    rm -rf bin/
    rm -rf frontend/dist/
    rm -rf frontend/node_modules/
    echo -e "${GREEN}Clean completed successfully.${NC}"
fi

# 2. Build Frontend
if [ "$BUILD_FRONTEND" = true ]; then
    echo -e "${YELLOW}Building Frontend Assets...${NC}"
    cd frontend
    if [ ! -d "node_modules" ]; then
        echo "node_modules not found. Installing package dependencies..."
        npm install
    fi
    echo "Compiling frontend assets..."
    npm run build
    cd ..
    echo -e "${GREEN}Frontend assets built successfully.${NC}"
fi

# 3. Build Go Backend
if [ "$BUILD_BACKEND" = true ]; then
    echo -e "${YELLOW}Building Go Backend Binary...${NC}"
    
    # Ensure frontend/dist exists, since Go embeds it
    if [ ! -d "frontend/dist" ]; then
        echo -e "${RED}Error: frontend/dist not found. The Go backend embeds frontend assets, so the frontend must be built first.${NC}"
        echo -e "${YELLOW}Hint: Run the build script without -b/--backend to build both, or run with -f/--frontend first.${NC}"
        exit 1
    fi
    
    mkdir -p bin
    go build -o bin/nexus-research main.go
    echo -e "${GREEN}Go backend binary compiled successfully.${NC}"
fi

# 4. Run tests if requested
if [ "$RUN_TESTS" = true ]; then
    echo -e "${YELLOW}Running tests...${NC}"
    go test -coverprofile=coverage.out ./db/... ./cmd/... ./server/... ./tui/...
    echo -e "${GREEN}All tests passed successfully.${NC}"
fi

echo -e "${BLUE}=============================================${NC}"
echo -e "${GREEN}  BUILD SUCCESSFUL${NC}"
if [ "$BUILD_BACKEND" = true ]; then
    echo -e "  Binary located at: ${GREEN}bin/nexus-research${NC}"
fi
echo -e "${BLUE}=============================================${NC}"
