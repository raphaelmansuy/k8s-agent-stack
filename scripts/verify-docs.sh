#!/bin/bash

# Documentation Link Verification Script
# Copyright 2025 Raphaël MANSUY
# Licensed under the Apache License, Version 2.0

set -e

echo "🔍 k8s-agent-stack Documentation Verification"
echo "=============================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
total_files=0
files_with_navigation=0
missing_navigation=()
broken_links=()

# Function to check if file has navigation footer
check_navigation() {
    local file=$1
    if grep -q "Back to\|← Back to" "$file"; then
        return 0
    else
        return 1
    fi
}

# Function to extract and verify internal markdown links
check_internal_links() {
    local file=$1
    local dir=$(dirname "$file")
    
    # Extract markdown links
    grep -oE '\[([^]]+)\]\(([^)]+\.md[^)]*)\)' "$file" | while read -r link; do
        # Extract the path from the link
        local path=$(echo "$link" | sed -E 's/.*\]\(([^)]+)\).*/\1/' | cut -d'#' -f1)
        
        # Skip external links
        if [[ "$path" =~ ^http ]]; then
            continue
        fi
        
        # Resolve relative path
        if [[ "$path" =~ ^\.\. ]]; then
            local target="$dir/$path"
        elif [[ "$path" =~ ^\. ]]; then
            local target="$dir/$path"
        else
            local target="$dir/$path"
        fi
        
        # Normalize path
        target=$(realpath -m "$target" 2>/dev/null || echo "$target")
        
        # Check if file exists
        if [ ! -f "$target" ]; then
            echo "❌ Broken link in $file: $path → $target"
            broken_links+=("$file: $path")
        fi
    done
}

echo "1️⃣  Checking all markdown files..."
echo ""

# Find all markdown files (excluding node_modules and similar)
while IFS= read -r file; do
    ((total_files++))
    
    # Skip certain files
    if [[ "$file" =~ node_modules|\.git|archive/logs ]]; then
        continue
    fi
    
    # Check for navigation footer (only for docs, examples, and agent READMEs)
    if [[ "$file" =~ docs/.*\.md$ ]] || \
       [[ "$file" =~ examples/README\.md$ ]] || \
       [[ "$file" =~ kagent-adk-agent/README\.md$ ]] || \
       [[ "$file" =~ CONTRIBUTING\.md$ ]] || \
       [[ "$file" =~ archive/README\.md$ ]]; then
        
        if check_navigation "$file"; then
            ((files_with_navigation++))
            echo -e "${GREEN}✓${NC} Navigation found: $file"
        else
            missing_navigation+=("$file")
            echo -e "${RED}✗${NC} Missing navigation: $file"
        fi
    fi
    
    # Check internal links
    check_internal_links "$file"
    
done < <(find . -name "*.md" -type f | grep -v node_modules | sort)

echo ""
echo "2️⃣  Summary"
echo "============"
echo ""
echo "Total markdown files checked: $total_files"
echo "Files with navigation footers: $files_with_navigation"
echo ""

if [ ${#missing_navigation[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Files missing navigation:${NC}"
    for file in "${missing_navigation[@]}"; do
        echo "   - $file"
    done
    echo ""
fi

if [ ${#broken_links[@]} -gt 0 ]; then
    echo -e "${RED}❌ Broken internal links found:${NC}"
    for link in "${broken_links[@]}"; do
        echo "   - $link"
    done
    echo ""
    exit 1
else
    echo -e "${GREEN}✅ No broken internal links found!${NC}"
    echo ""
fi

echo "3️⃣  Required Documents Check"
echo "============================"
echo ""

required_docs=(
    "README.md"
    "CONTRIBUTING.md"
    "CONTRIBUTORS.md"
    "LICENSE"
    "docs/README.md"
    "docs/DOCUMENTATION_INDEX.md"
    "docs/getting-started.md"
    "docs/architecture.md"
    "docs/deployment-guide.md"
    "docs/troubleshooting.md"
    "docs/glossary.md"
    "docs/tool-installation.md"
    "docs/building-google-adk-agents-for-kagent.md"
    "docs/kagent-adk-a2a-architecture.md"
    "examples/README.md"
    "kagent-adk-agent/README.md"
)

all_present=true
for doc in "${required_docs[@]}"; do
    if [ -f "$doc" ]; then
        echo -e "${GREEN}✓${NC} $doc"
    else
        echo -e "${RED}✗${NC} Missing: $doc"
        all_present=false
    fi
done

echo ""

if $all_present && [ ${#missing_navigation[@]} -eq 0 ] && [ ${#broken_links[@]} -eq 0 ]; then
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ All documentation checks passed!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    exit 0
else
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}⚠️  Some documentation issues found${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    exit 1
fi
