# Kagent Portal UI - Access Guide

**Last Updated**: 2025-12-16  
**Status**: Ready for Deployment

## What is Kagent Portal?

**Kagent Portal** is a unified web-based dashboard that provides:
- ✅ **Agent Management**: View, configure, and manage all deployed agents
- ✅ **Interactive Chat**: Chat with any agent through a web UI
- ✅ **Agent Monitoring**: Track agent status, logs, and performance
- ✅ **Tool Management**: Browse and manage available tools/functions
- ✅ **Agent Deployment**: Deploy and update agents through the UI
- ✅ **Configuration Management**: Manage agent settings and environment variables

Think of it as the **control panel** for your entire agent infrastructure.

---

## Quick Start: Access Kagent Portal

### Option 1: Local Port-Forward (Recommended for Development)

```bash
# Port-forward to local machine
kubectl port-forward -n kagent svc/kagent-web 3000:3000

# Open in browser
open http://localhost:3000
```

**Expected Output:**
```
Forwarding from 127.0.0.1:3000 -> 3000
Forwarding from [::1]:3000 -> 3000
```

**Access**: Open your browser to `http://localhost:3000`

### Option 2: Knative Service External URL (Full Deployment)

```bash
# Get the external URL (if deployed as Knative Service)
kubectl get ksvc kagent-portal -n kagent -o jsonpath='{.status.url}'

# Access directly (requires sslip.io DNS or custom domain)
curl http://kagent-portal.kagent.192.168.139.2.sslip.io
```

### Option 3: Port-Forward Service (ClusterIP)

```bash
# Direct service port-forward
kubectl port-forward -n kagent svc/kagent-web 3000:3000

# From another terminal:
curl http://localhost:3000
```

---

## Installation & Deployment

### Step 1: Deploy Kagent Portal

```bash
# Deploy the portal
kubectl apply -f kagent-portal.yaml

# Verify deployment
kubectl get pods -n kagent -l app=kagent-web
kubectl get svc -n kagent -l app=kagent-web
```

**Expected Output:**
```
NAME                          READY   STATUS    RESTARTS   AGE
kagent-web-5d4c7f6d-abc123   1/1     Running   0          2m

NAME           TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
kagent-web     ClusterIP   192.168.194.xxx  <none>        3000/TCP   2m
```

### Step 2: Enable Port-Forward

```bash
# Start port-forward in background (or new terminal)
kubectl port-forward -n kagent svc/kagent-web 3000:3000 &

# Wait for it to be ready
sleep 2

# Test connectivity
curl http://localhost:3000
```

### Step 3: Open in Browser

Navigate to: **`http://localhost:3000`**

---

## Portal Features & Navigation

### Dashboard Overview

```
┌─────────────────────────────────────────────────────┐
│  Kagent Portal - Agent Management Dashboard         │
├─────────────────────────────────────────────────────┤
│                                                     │
│ 📊 Dashboard                                       │
│  ├─ System Status (Kubernetes cluster info)        │
│  ├─ Running Agents (count & status)                │
│  ├─ Resource Usage (CPU, Memory)                   │
│  └─ Recent Activity                                │
│                                                     │
│ 🤖 Agents                                          │
│  ├─ List all deployed agents                       │
│  ├─ View agent status & logs                       │
│  ├─ Chat interface for each agent                  │
│  ├─ Deploy new agents                              │
│  └─ Configure agent settings                       │
│                                                     │
│ 🛠️ Tools                                            │
│  ├─ Browse available tools                         │
│  ├─ View tool documentation                        │
│  ├─ Test tools                                     │
│  └─ Manage tool servers                            │
│                                                     │
│ ⚙️ Settings                                         │
│  ├─ API configuration                              │
│  ├─ Authentication settings                        │
│  ├─ Agent defaults                                 │
│  └─ System configuration                           │
│                                                     │
│ 📋 Logs & Monitoring                               │
│  ├─ Agent logs                                     │
│  ├─ System events                                  │
│  ├─ Performance metrics                            │
│  └─ Debugging info                                 │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Chat with Agents

1. **Navigate to Agents section**
2. **Select an agent** (e.g., `google-adk-agent`)
3. **Open Chat interface**
4. **Type your message** and press Send
5. **See real-time responses** with tool calls visualization

Example:
```
You: "What time is it?"
Agent: [Processing...]
Agent: "It is 10:35 AM PST on December 16, 2025"
```

### View Agent Status

```bash
# In Portal:
1. Agents → [Agent Name]
2. View Status, Logs, Configuration
3. See live metrics and performance
```

### Deploy New Agent

```bash
# In Portal:
1. Click "+ Deploy Agent"
2. Upload YAML manifest
3. Configure settings
4. Click Deploy
5. Monitor deployment progress
```

---

## Make Commands Integration

### Deploy Portal with Make

```bash
# Add to Makefile (if not exists):
.PHONY: portal-deploy
portal-deploy:
	kubectl apply -f kagent-portal.yaml

.PHONY: portal-access
portal-access:
	@echo "Opening Kagent Portal..."
	@echo "URL: http://localhost:3000"
	@echo "Starting port-forward..."
	kubectl port-forward -n kagent svc/kagent-web 3000:3000

.PHONY: portal-status
portal-status:
	kubectl get pods -n kagent -l app=kagent-web
	kubectl get svc -n kagent -l app=kagent-web
```

### Usage

```bash
make portal-deploy    # Deploy the portal
make portal-access    # Start port-forward
make portal-status    # Check status
```

---

## Accessing Your Agents Through Portal

### View Your Google ADK Agent

1. **Open Portal**: `http://localhost:3000`
2. **Go to Agents**: Click "Agents" in sidebar
3. **Find Agent**: Look for `google-adk-agent`
4. **Click on it**: View agent details
5. **Chat Tab**: Open the chat interface
6. **Send Message**: Type and send to your agent

### Example Interaction

```
Portal Chat Interface:
═══════════════════════════════════════════

User Input:
  "What is 2 + 2?"

Agent Response:
  [Tool Calls]
  ├─ calculator.add(2, 2)
  └─ Returns: 4
  
  Answer: "2 + 2 equals 4"

[Show more] [Copy] [Export]
```

---

## Portal Architecture

```
┌──────────────────────────────────────────────────────────┐
│                   User Browser                           │
│          (http://localhost:3000)                         │
└────────────────────┬─────────────────────────────────────┘
                     │ HTTP/WebSocket
        ┌────────────▼───────────┐
        │  Kagent Web UI         │
        │  (React/Next.js)       │
        │  Port: 3000            │
        └────────────┬───────────┘
                     │
        ┌────────────▼─────────────────┐
        │  Kagent Server API           │
        │  Port: 8083 (in-cluster)     │
        └────────────┬─────────────────┘
                     │
        ┌────────────▼──────────────────────────┐
        │  Kubernetes Kagent Controller         │
        │  - Manages Agent CRDs                 │
        │  - Orchestrates deployments           │
        │  - Handles agent lifecycle            │
        └────────────┬──────────────────────────┘
                     │
        ┌────────────▼──────────────────────────┐
        │  Agent Pods in kagent Namespace       │
        │  - google-adk-agent                   │
        │  - Custom agents                      │
        │  - Tool servers                       │
        └───────────────────────────────────────┘
```

---

## Access Methods Comparison

| Method | URL | Setup | Use Case | Speed |
|--------|-----|-------|----------|-------|
| **Port-Forward** | localhost:3000 | 1 command | Development | Fast |
| **Knative Service** | sslip.io URL | Full deploy | Production | Medium |
| **ClusterIP** | Internal DNS | Inside cluster | Admin | Fast |
| **Ingress** | Custom domain | Advanced setup | Enterprise | Medium |

---

## Troubleshooting Portal Access

### Problem: "Connection refused"

```bash
# Solution 1: Check if portal is running
kubectl get pods -n kagent -l app=kagent-web

# Solution 2: Verify service exists
kubectl get svc kagent-web -n kagent

# Solution 3: Check port-forward
ps aux | grep port-forward
```

### Problem: "Portal shows 'Connecting to API...'"

```bash
# The portal needs the Kagent server API running
# Check if other components are deployed:
kubectl get all -n kagent

# Verify kagent namespace is healthy:
make verify
```

### Problem: "No agents showing in portal"

```bash
# Verify agents are deployed:
kubectl get pods -n kagent

# Check agent CRDs:
kubectl get agents -n kagent 2>/dev/null || \
  echo "No agents CRD found - deploy agents first"

# Check logs:
make agent-logs
```

### Problem: "Can't reach from external machine"

```bash
# Port-forward only works from localhost
# Solution 1: Use Knative service (deploy kagent-portal-knative.yaml)
# Solution 2: Use ngrok to expose locally
ngrok http 3000
```

---

## Complete Access Workflow

### 1. Deploy Portal
```bash
kubectl apply -f kagent-portal.yaml
kubectl get pods -n kagent -l app=kagent-web
```

### 2. Port-Forward to Local
```bash
kubectl port-forward -n kagent svc/kagent-web 3000:3000 &
```

### 3. Open Browser
```bash
# Manually or via:
open http://localhost:3000
```

### 4. Explore Agents
- Click **Agents** in sidebar
- Click on **google-adk-agent**
- Click **Chat** tab
- Start chatting!

### 5. Monitor Status
- Dashboard shows real-time status
- View logs in **Logs** section
- Monitor metrics in **Metrics**

---

## Features by Agent

### Google ADK Agent (`google-adk-agent`)

**Available Through Portal:**
- ✅ Interactive chat
- ✅ View configuration
- ✅ Monitor pod status
- ✅ View logs
- ✅ See running metrics

**Chat Example:**
```
You: "Hello"
Agent: "Hello! I'm an AI agent running on Knative. How can I help?"
```

### Custom Agents

**Once Deployed:**
1. They appear in portal automatically
2. Get their own chat interface
3. Show logs and status
4. Configurable through portal

---

## Security Considerations

### Access Control
- Portal runs in kagent namespace
- Protected by Kubernetes RBAC
- Service account with minimal permissions

### Production Deployment
```bash
# Use authentication:
kubectl apply -f kagent-portal-secure.yaml

# or with TLS:
kubectl apply -f kagent-portal-tls.yaml
```

### Firewall Rules
```bash
# Limit access to port-forward from localhost only
# Use VPN/SSH tunnel for remote access
ssh -L 3000:localhost:3000 user@host
```

---

## Performance & Scaling

### Resource Usage
- CPU: 100m - 500m typical
- Memory: 256Mi - 512Mi typical
- Scales to 3 replicas under load

### Optimization Tips
1. **Use Knative Service** for auto-scaling
2. **Enable caching** in browser DevTools
3. **Use dashboard** instead of logs tab for monitoring
4. **Batch operations** in UI

---

## Automation

### Shell Alias for Quick Access

```bash
# Add to ~/.zshrc or ~/.bashrc:
alias kagent-portal='kubectl port-forward -n kagent svc/kagent-web 3000:3000 && open http://localhost:3000'

# Usage:
kagent-portal
```

### Script to Deploy & Access

```bash
#!/bin/bash
# deploy-kagent-portal.sh

set -e

echo "Deploying Kagent Portal..."
kubectl apply -f kagent-portal.yaml

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=ready pod -l app=kagent-web -n kagent --timeout=300s

echo "Starting port-forward..."
kubectl port-forward -n kagent svc/kagent-web 3000:3000 &
PF_PID=$!

echo "Opening browser..."
sleep 2
open http://localhost:3000 || xdg-open http://localhost:3000

echo "Portal is running. Press Ctrl+C to stop."
wait $PF_PID
```

---

## Integration with Makefile

Add these targets to your [Makefile](Makefile):

```makefile
.PHONY: portal-deploy
portal-deploy:
	@echo "Deploying Kagent Portal..."
	kubectl apply -f kagent-portal.yaml
	@echo "✓ Portal deployment manifest applied"

.PHONY: portal-access
portal-access:
	@echo "Starting Kagent Portal access..."
	@echo "Portal will be available at: http://localhost:3000"
	@echo "Press Ctrl+C to stop"
	kubectl port-forward -n kagent svc/kagent-web 3000:3000

.PHONY: portal-status
portal-status:
	@echo "Kagent Portal Status:"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@kubectl get pods -n kagent -l app=kagent-web
	@echo ""
	@echo "Services:"
	@kubectl get svc -n kagent -l app=kagent-web
	@echo ""
	@echo "Access URL: http://localhost:3000 (requires port-forward)"

.PHONY: portal-logs
portal-logs:
	kubectl logs -n kagent -l app=kagent-web -f

.PHONY: portal-clean
portal-clean:
	@echo "Removing Kagent Portal..."
	kubectl delete -f kagent-portal.yaml --ignore-not-found
	@echo "✓ Portal removed"
```

---

## Quick Reference

### Essential Commands

```bash
# Deploy
make portal-deploy

# Access
make portal-access

# Status
make portal-status

# Logs
make portal-logs

# Cleanup
make portal-clean
```

### URLs

| Service | URL | Context |
|---------|-----|---------|
| Portal UI | `http://localhost:3000` | After port-forward |
| Knative | `http://kagent-portal.kagent.192.168.139.2.sslip.io` | External |
| API | `http://kagent-server:8083` | Internal cluster |

---

## Next Steps

1. ✅ **Deploy Portal**: `make portal-deploy`
2. ✅ **Access Portal**: `make portal-access`
3. ✅ **Explore Agents**: Navigate to Agents section
4. ✅ **Chat with Agent**: Test google-adk-agent
5. ✅ **Deploy Custom Agent**: Use portal to deploy

---

## Support

**Issues?** Check:
1. `make portal-status` - Verify deployment
2. `make portal-logs` - View error logs
3. `kubectl get pods -n kagent` - Check cluster
4. Try port-forward manually: `kubectl port-forward -n kagent svc/kagent-web 3000:3000`

**Documentation:**
- [Building Google ADK Agents for Kagent](docs/building-google-adk-agents-for-kagent.md)
- [SETUP.md](SETUP.md)
- [DEPLOYMENT_COMPLETE.md](DEPLOYMENT_COMPLETE.md)

---

**Kagent Portal Ready!** 🚀
