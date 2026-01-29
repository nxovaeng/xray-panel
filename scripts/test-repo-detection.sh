#!/bin/bash

# Test script for GitHub repository detection

echo "Testing GitHub repository detection..."
echo ""

# Function to detect GitHub repository
detect_github_repo() {
    local repo=""
    
    # 1. Check environment variable
    if [[ -n "$GITHUB_REPO" ]]; then
        repo="$GITHUB_REPO"
        echo "  Source: Environment variable"
    # 2. Try to detect from git remote
    elif command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
        repo=$(git config --get remote.origin.url 2>/dev/null | sed -E 's#.*github\.com[:/]([^/]+/[^/]+)(\.git)?$#\1#')
        if [[ -n "$repo" ]]; then
            echo "  Source: Git remote"
        fi
    fi
    
    # 3. Fallback to default
    if [[ -z "$repo" ]]; then
        repo="nxovaeng/xray-panel"
        echo "  Source: Default fallback"
    fi
    
    echo "$repo"
}

# Test 1: Default (no environment variable, no git)
echo "Test 1: Default detection"
cd /tmp
DETECTED_REPO=$(detect_github_repo)
echo "  Result: $DETECTED_REPO"
echo ""

# Test 2: With environment variable
echo "Test 2: Environment variable"
cd /tmp
GITHUB_REPO="custom-user/custom-repo" DETECTED_REPO=$(detect_github_repo)
echo "  Result: $DETECTED_REPO"
echo ""

# Test 3: From git remote (if in git repo)
echo "Test 3: Git remote detection"
if git rev-parse --git-dir > /dev/null 2>&1; then
    DETECTED_REPO=$(detect_github_repo)
    echo "  Result: $DETECTED_REPO"
else
    echo "  Skipped: Not in a git repository"
fi
echo ""

# Test 4: Various GitHub URL formats
echo "Test 4: URL parsing tests"
test_urls=(
    "https://github.com/user/repo.git"
    "git@github.com:user/repo.git"
    "https://github.com/user/repo"
    "git@github.com:user/repo"
)

for url in "${test_urls[@]}"; do
    parsed=$(echo "$url" | sed -E 's#.*github\.com[:/]([^/]+/[^/]+)(\.git)?$#\1#')
    echo "  $url"
    echo "    -> $parsed"
done

echo ""
echo "All tests completed!"
