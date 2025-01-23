#!/bin/bash

# Exit on any error
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if we're on main branch
current_branch=$(git branch --show-current)
if [ "$current_branch" != "main" ]; then
    echo -e "${RED}Error: Must be on main branch to release${NC}"
    exit 1
fi

# Check if working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}Error: Working directory is not clean. Commit or stash changes first.${NC}"
    exit 1
fi

# Get the current version
current_version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
current_version=${current_version#v} # Remove leading 'v' if present

# Increment the patch version
major=$(echo $current_version | cut -d. -f1)
minor=$(echo $current_version | cut -d. -f2)
patch=$(echo $current_version | cut -d. -f3)
new_patch=$((patch + 1))
new_version="${major}.${minor}.${new_patch}"

# Validate version format
if ! [[ $new_version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Invalid version format: $new_version${NC}"
    exit 1
fi

echo -e "${GREEN}Current version: v$current_version${NC}"
echo -e "${GREEN}New version: v$new_version${NC}"

# Run go mod tidy to ensure dependencies are up to date
echo -e "${YELLOW}Running go mod tidy...${NC}"
go mod tidy

# Add all changes
git add .

# Commit changes with a better format
echo "Enter commit message (press enter to use 'chore: bump version to v$new_version'):"
read commit_msg
if [ -z "$commit_msg" ]; then
    commit_msg="chore: bump version to v$new_version"
fi
git commit -m "$commit_msg"

# Create and push new tag
echo -e "${YELLOW}Creating and pushing new tag...${NC}"
git tag "v$new_version"
git push origin main || { echo -e "${RED}Failed to push to main${NC}"; exit 1; }
git push origin "v$new_version" || { echo -e "${RED}Failed to push tag${NC}"; exit 1; }

echo -e "${GREEN}Package updated and pushed successfully!${NC}"
echo -e "${GREEN}New version v$new_version is now available${NC}"

# Create a temporary directory for registering with proxy.golang.org
echo -e "${YELLOW}Registering package with proxy.golang.org...${NC}"
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT  # Ensure cleanup even if script fails

cd "$temp_dir"
if ! go mod init temp; then
    echo -e "${RED}Failed to initialize temporary module${NC}"
    exit 1
fi

if ! GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/awade12/go-db@v$new_version; then
    echo -e "${RED}Failed to register with proxy.golang.org${NC}"
    exit 1
fi

cd - > /dev/null

# Make a direct request to proxy.golang.org
echo -e "${YELLOW}Making direct request to proxy.golang.org...${NC}"
if ! curl -s "https://proxy.golang.org/github.com/awade12/go-db/@v/v$new_version.info" > /dev/null; then
    echo -e "${RED}Failed to verify package on proxy.golang.org${NC}"
    exit 1
fi

echo -e "${GREEN}Package registered with proxy.golang.org${NC}"
echo ""
echo -e "${GREEN}To update the package on other machines, run:${NC}"
echo -e "${YELLOW}go install github.com/awade12/go-db@v$new_version${NC}"
