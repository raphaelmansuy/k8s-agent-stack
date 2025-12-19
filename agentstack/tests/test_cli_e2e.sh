#!/bin/bash

# AgentStack CLI E2E Test Script
# This script tests all agentctl commands against a running API.

set -e

# Configuration
API_URL=${API_URL:-"http://localhost:8080"}
API_KEY=${API_KEY:-"test-api-key"}

# Clean API_KEY (remove newlines/spaces)
API_KEY=$(echo "$API_KEY" | tr -d '\n' | tr -d ' ')

CLI_BIN="./bin/agentctl"
TEST_CONFIG="/tmp/agentctl-test-config.yaml"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Starting AgentStack CLI E2E Tests..."
echo "API URL: $API_URL"
echo "Config: $TEST_CONFIG"

# Ensure CLI is built
if [ ! -f "$CLI_BIN" ]; then
    echo "Building agentctl..."
    go build -o bin/agentctl ./cli/cmd/agentctl
fi

# Helper function to run a command and check its exit code
run_test() {
    local name=$1
    local cmd=$2
    echo -n "Testing $name... "
    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}PASSED${NC}"
    else
        echo -e "${RED}FAILED${NC}"
        echo "Command: $cmd"
        exit 1
    fi
}

# Cleanup on exit
cleanup() {
    # rm -f "$TEST_CONFIG"
    echo "Cleanup skipped"
}
trap cleanup EXIT

# 1. Version and Help
run_test "version" "$CLI_BIN version"
run_test "help" "$CLI_BIN --help"

# 2. Config and Login
run_test "config view" "$CLI_BIN config view --config $TEST_CONFIG"
run_test "login" "$CLI_BIN login --endpoint $API_URL --api-key $API_KEY --config $TEST_CONFIG"

# 3. Project Management
RANDOM_ID=$(date +%s | tail -c 4)
PROJECT_NAME="E2E Project $RANDOM_ID"
run_test "project create" "$CLI_BIN project create --name '$PROJECT_NAME' --description 'Created by E2E test' --config $TEST_CONFIG"
run_test "project list" "$CLI_BIN project list --config $TEST_CONFIG"
# Get the project ID from the list
PROJECT_ID=$($CLI_BIN project list --config $TEST_CONFIG -o json | jq -r ".[] | select(.name == \"$PROJECT_NAME\") | .id")
if [ -z "$PROJECT_ID" ] || [ "$PROJECT_ID" == "null" ]; then
    echo -e "${RED}FAILED: Could not find project ID for $PROJECT_NAME${NC}"
    exit 1
fi
run_test "project get" "$CLI_BIN project get $PROJECT_ID --config $TEST_CONFIG"

# 4. Agent Management
run_test "agent create" "$CLI_BIN agent create --name 'E2E Agent' --project $PROJECT_ID --config $TEST_CONFIG"
run_test "agent list" "$CLI_BIN agent list --project $PROJECT_ID --config $TEST_CONFIG"
AGENT_ID=$($CLI_BIN agent list --project $PROJECT_ID --config $TEST_CONFIG -o json | jq -r '.[0].id')
run_test "agent get" "$CLI_BIN agent get $AGENT_ID --config $TEST_CONFIG"

# 5. Declarative Apply
run_test "apply" "$CLI_BIN apply -f tests/e2e_manifest.yaml --config $TEST_CONFIG"

# 6. Deployment Management
run_test "deploy list" "$CLI_BIN deploy list --config $TEST_CONFIG"

# --- Real KAgent Deployment Section ---
# Documentation: How to build the Docker image for AgentStack
# To deploy a real agent, you first need to build its container image:
# 1. cd kagent-adk-agent
# 2. docker build -t kagent-adk-agent:latest .
# 3. (Optional) Push to a registry if deploying to a remote cluster

KAGENT_IMAGE="kagent-adk-agent:latest"
if command -v docker >/dev/null 2>&1; then
    echo "Building real KAgent image..."
    (cd ../kagent-adk-agent && docker build -t $KAGENT_IMAGE .)
else
    echo "Docker not found, skipping real image build. Using placeholder."
    KAGENT_IMAGE="nginx"
fi

# Create a deployment with the real agent image
run_test "deploy create" "$CLI_BIN deploy create --name 'E2E-KAgent' --image '$KAGENT_IMAGE' --config $TEST_CONFIG"
DEPLOY_ID=$($CLI_BIN deploy list --config $TEST_CONFIG -o json | jq -r '.[] | select(.name == "E2E-KAgent") | .id')
if [ -z "$DEPLOY_ID" ] || [ "$DEPLOY_ID" == "null" ]; then
    # Fallback to first deployment if name matching fails
    DEPLOY_ID=$($CLI_BIN deploy list --config $TEST_CONFIG -o json | jq -r '.[0].id')
fi

run_test "deploy status" "$CLI_BIN deploy status $DEPLOY_ID --config $TEST_CONFIG"
run_test "deploy scale" "$CLI_BIN deploy scale $DEPLOY_ID --replicas 2 --config $TEST_CONFIG"
run_test "deploy restart" "$CLI_BIN deploy restart $DEPLOY_ID --config $TEST_CONFIG"
run_test "deploy delete" "$CLI_BIN deploy delete $DEPLOY_ID --force --config $TEST_CONFIG"

# 7. API Key Management
run_test "keys list" "$CLI_BIN keys list --config $TEST_CONFIG"
run_test "keys create" "$CLI_BIN keys create --name 'E2E Key' --scopes 'agent:read' --config $TEST_CONFIG"
KEY_ID=$($CLI_BIN keys list --config $TEST_CONFIG -o json | jq -r '.[0].id')
run_test "keys rotate" "$CLI_BIN keys rotate $KEY_ID --force --config $TEST_CONFIG"
run_test "keys delete" "$CLI_BIN keys delete $KEY_ID --config $TEST_CONFIG"

# 8. Logs
# Note: Logs might be empty but the command should still succeed
run_test "logs agent" "$CLI_BIN logs $AGENT_ID --tail 10 --config $TEST_CONFIG"
run_test "logs deploy" "$CLI_BIN logs deploy/$DEPLOY_ID --tail 10 --config $TEST_CONFIG"

# 9. Chat (Non-interactive check)
# We can't easily test interactive chat in a script, but we can check if it starts or shows help
run_test "chat help" "$CLI_BIN chat --help"

echo -e "\n${GREEN}All CLI E2E tests passed successfully!${NC}"
