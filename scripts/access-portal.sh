#!/bin/bash
#
# Copyright 2025 Raphaël MANSUY
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#
# Kagent Portal Quick Access Script
# Opens the Kagent Portal UI in your browser for agent management
#

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Kagent Portal - Agent Management Dashboard${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

# Check if portal is deployed
echo -e "${YELLOW}1. Checking portal deployment...${NC}"
POD_COUNT=$(kubectl get pods -n kagent -l app=kagent-web --no-headers 2>/dev/null | wc -l)

if [ "$POD_COUNT" -eq 0 ]; then
  echo -e "${RED}✗ Portal not deployed${NC}"
  echo ""
  echo -e "${YELLOW}Deploying Kagent Portal...${NC}"
  kubectl apply -f deploy/kagent-portal.yaml
  echo -e "${GREEN}✓ Portal deployed$(NC)"
  echo ""
  echo "Waiting for pod to be ready..."
  kubectl wait --for=condition=ready pod -l app=kagent-web -n kagent --timeout=60s 2>/dev/null || true
else
  POD_STATUS=$(kubectl get pods -n kagent -l app=kagent-web -o jsonpath='{.items[0].status.phase}')
  if [ "$POD_STATUS" = "Running" ]; then
    echo -e "${GREEN}✓ Portal is running${NC}"
  else
    echo -e "${YELLOW}⚠ Portal status: $POD_STATUS${NC}"
  fi
fi

echo ""
echo -e "${YELLOW}2. Starting port-forward...${NC}"

# Kill any existing port-forwards to port 3000
pkill -f "port-forward.*3000" 2>/dev/null || true
sleep 1

# Start port-forward
kubectl port-forward -n kagent svc/kagent-web 3000:3000 &
PF_PID=$!
sleep 2

# Verify port-forward is working
if curl -s http://localhost:3000 > /dev/null; then
  echo -e "${GREEN}✓ Port-forward successful${NC}"
else
  echo -e "${RED}✗ Port-forward failed${NC}"
  exit 1
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ Kagent Portal Ready!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}📍 Portal URL: ${GREEN}http://localhost:3000${NC}"
echo ""
echo "Features:"
echo "  ✓ Agent Management Dashboard"
echo "  ✓ Active Agents Overview"
echo "  ✓ Quick Access Commands"
echo "  ✓ System Components Info"
echo "  ✓ Documentation Links"
echo ""
echo -e "${YELLOW}Opening in browser...${NC}"

# Try to open in browser
if command -v open > /dev/null; then
  open http://localhost:3000
elif command -v xdg-open > /dev/null; then
  xdg-open http://localhost:3000 &
else
  echo -e "${YELLOW}Please open http://localhost:3000 in your browser${NC}"
fi

echo ""
echo -e "${YELLOW}Press Ctrl+C to stop the port-forward${NC}"
wait $PF_PID
