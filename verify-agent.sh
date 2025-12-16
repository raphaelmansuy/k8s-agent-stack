#!/bin/bash
#
# Comprehensive verification script for kagent + ADK Agent setup
#

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  kagent + ADK Agent Verification${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

# 1. Check Kubernetes connectivity
echo -e "${YELLOW}1. Checking Kubernetes connectivity...${NC}"
if kubectl cluster-info >/dev/null 2>&1; then
  echo -e "   ${GREEN}✓ Kubernetes cluster connected${NC}"
else
  echo -e "   ${RED}✗ Kubernetes cluster not found${NC}"
  exit 1
fi

# 2. Check kagent namespace
echo -e "${YELLOW}2. Checking kagent namespace...${NC}"
if kubectl get ns kagent >/dev/null 2>&1; then
  echo -e "   ${GREEN}✓ kagent namespace exists${NC}"
else
  echo -e "   ${RED}✗ kagent namespace not found${NC}"
  exit 1
fi

# 3. Check OpenAI secret
echo -e "${YELLOW}3. Checking OpenAI API secret...${NC}"
if kubectl get secret openai-api-key -n kagent >/dev/null 2>&1; then
  echo -e "   ${GREEN}✓ OpenAI API key secret found${NC}"
else
  echo -e "   ${RED}✗ OpenAI API key secret not found${NC}"
  exit 1
fi

# 4. Check Knative service
echo -e "${YELLOW}4. Checking ADK Agent Knative service...${NC}"
AGENT_STATUS=$(kubectl get ksvc google-adk-agent -n kagent -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
if [ "$AGENT_STATUS" = "True" ]; then
  echo -e "   ${GREEN}✓ ADK Agent service is Ready${NC}"
else
  REASON=$(kubectl get ksvc google-adk-agent -n kagent -o jsonpath='{.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null || echo "Unknown")
  echo -e "   ${YELLOW}⚠ Service status: $REASON (may still be initializing)${NC}"
fi

# 5. Check pods
echo -e "${YELLOW}5. Checking ADK Agent pods...${NC}"
POD_COUNT=$(kubectl get pods -n kagent --no-headers 2>/dev/null | wc -l)
if [ "$POD_COUNT" -gt 0 ]; then
  echo -e "   ${GREEN}✓ Agent pods running ($POD_COUNT)${NC}"
  kubectl get pods -n kagent --no-headers | head -5 | sed 's/^/     /'
else
  echo -e "   ${RED}✗ No agent pods found${NC}"
  exit 1
fi

# 6. Check services
echo -e "${YELLOW}6. Checking services...${NC}"
if kubectl get svc google-adk-agent-00001-private -n kagent >/dev/null 2>&1; then
  SVC_IP=$(kubectl get svc google-adk-agent-00001-private -n kagent -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
  echo -e "   ${GREEN}✓ Private service found (IP: $SVC_IP)${NC}"
else
  echo -e "   ${RED}✗ Service not found${NC}"
  exit 1
fi

# 7. Check URL
echo -e "${YELLOW}7. Checking external URL...${NC}"
AGENT_URL=$(kubectl get ksvc google-adk-agent -n kagent -o jsonpath='{.status.url}' 2>/dev/null || echo "")
if [ -n "$AGENT_URL" ]; then
  echo -e "   ${GREEN}✓ Agent URL: $AGENT_URL${NC}"
else
  echo -e "   ${YELLOW}⚠ URL not yet assigned${NC}"
fi

# 8. Test connectivity via port-forward
echo -e "${YELLOW}8. Testing port-forward connectivity...${NC}"
POD=$(kubectl get pods -n kagent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$POD" ]; then
  # Start port-forward in background
  kubectl port-forward -n kagent svc/google-adk-agent-00001-private 8080:80 >/tmp/pf-test.log 2>&1 &
  PF_PID=$!
  sleep 2
  
  # Test endpoint
  if curl -sS --max-time 5 http://localhost:8080 >/dev/null 2>&1; then
    echo -e "   ${GREEN}✓ Agent is reachable via localhost:8080${NC}"
  else
    echo -e "   ${YELLOW}⚠ Port-forward failed (pod may still be initializing)${NC}"
  fi
  
  # Kill port-forward
  kill $PF_PID 2>/dev/null || true
  sleep 1
else
  echo -e "   ${YELLOW}⚠ No pods found for connectivity test${NC}"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ Verification Complete!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Available Commands:${NC}"
echo "  make agent-status      # Check agent status"
echo "  make port-forward      # Port-forward for local testing"
echo "  make agent-logs        # View agent logs"
echo "  make agent-describe    # Show pod details"
echo ""
echo -e "${YELLOW}Access Agent:${NC}"
echo "  Local:  make port-forward, then http://localhost:8080"
echo "  Remote: $AGENT_URL"
echo ""
