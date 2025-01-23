#!/bin/bash

# Exit on any error
set -e

# Get the current version
current_version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Increment the patch version
major=$(echo $current_version | cut -d. -f1)
minor=$(echo $current_version | cut -d. -f2)
patch=$(echo $current_version | cut -d. -f3)
new_patch=$((patch + 1))
new_version="${major}.${minor}.${new_patch}"

echo "Current version: $current_version"
echo "New version: $new_version"

# Run go mod tidy to ensure dependencies are up to date
go mod tidy

# Add all changes
git add .

# Commit changes
echo "Enter commit message (press enter to use 'update: version $new_version'):"
read commit_msg
if [ -z "$commit_msg" ]; then
    commit_msg="update: version $new_version"
fi
git commit -m "$commit_msg"

# Create and push new tag
git tag "v$new_version"
git push origin main
git push origin "v$new_version"

echo "Package updated and pushed successfully!"
echo "New version v$new_version is now available"

# Create a temporary directory for registering with proxy.golang.org
echo "Registering package with proxy.golang.org..."
temp_dir=$(mktemp -d)
cd "$temp_dir"
go mod init temp
GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/awade12/go-db@v$new_version
cd - > /dev/null
rm -rf "$temp_dir"

# Make a direct request to proxy.golang.org
echo "Making direct request to proxy.golang.org..."
curl -s "https://proxy.golang.org/github.com/awade12/go-db/@v/v$new_version.info" > /dev/null

echo "Package registered with proxy.golang.org"
echo ""
echo "To update the package on other machines, run:"
echo "go install github.com/awade12/go-db@v$new_version"
