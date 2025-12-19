# 🎯 Kagent UI — Quick Access & Setup

**Status**: ✅ **DEPLOYED & OPERATIONAL**  
**Last updated**: 2025-12-16

---

## Quick access (recommended)
Use the bundled `agentctl` which handles port-forwarding and A2A streaming:

```bash
# start UI and forward required ports
./agentstack/bin/agentctl ui --port 3000
# then open
open http://localhost:3000
```

Shortcut:
```bash
make agentstack-ui
```

---

## ✅ What's Deployed

### Kagent Complete Platform
- ✅ **kagent-ui** - Official web interface (running on port 8080)
- ✅ **kagent-controller** - Manages Kubernetes resources
- ✅ **kagent-engine** - AI agent runtime
- ✅ **kagent-kmcp-controller** - Manages MCP Server resources

### Pre-installed Agents (10 agents)
- ✅ argo-rollouts-agent
- ✅ cilium-debug-agent
- ✅ cilium-manager-agent
- ✅ cilium-policy-agent
- ✅ helm-agent
- ✅ istio-agent
- ✅ k8s-agent
- ✅ kgateway-agent
- ✅ observability-agent
- ✅ promql-agent

### Pre-installed MCP Tools
- ✅ kagent-tools (built-in tools)
- ✅ grafana-mcp
- ✅ querydoc

---

## 🎯 Kagent UI Features

The official Kagent UI provides:

### 1. **Agent Management Dashboard**
- View all deployed agents
- Monitor agent status
- Deploy new agents
- Configure agent settings
- View agent conversations

### 2. **Chat Interface**
- Interactive chat with agents
- Multi-agent conversations
- Agent-to-Agent (A2A) communication
- View conversation history
- Test agent responses

### 3. **Tool Management**
- Browse available MCP tools
- Configure tool servers
- Monitor tool usage
- Add custom tools

### 4. **Model Configuration**
- Configure LLM providers (OpenAI, Anthropic, Gemini, etc.)
- Set default models
- Manage API keys
- Configure model parameters

### 5. **Observability**
- View agent traces (OpenTelemetry)
- Monitor performance
- Debug agent behavior
- View logs and metrics

### 6. **Resource Management**
- Manage Kubernetes resources
- View agent deployments
- Configure memory stores
- Monitor resource usage

---

## 📖 Access Methods

**Recommended:** `agentctl` (handles port-forwarding + A2A streaming)

```bash
./agentstack/bin/agentctl ui --port 3000
open http://localhost:3000
# or
make agentstack-ui
```

**Manual (only if necessary)**
```bash
# UI
kubectl -n kagent port-forward service/kagent-ui 3000:80 &
# SSR/models (if models fail to load)
kubectl -n kagent port-forward service/kagent-ui 8080:8080 &
# (optional) A2A / websockets for streaming
kubectl -n kagent port-forward service/kagent-ui 8083:8083 &
```

**Production:** add an Ingress that routes to `service/kagent-ui:8080` and enable TLS.

---

## 🔧 Controller API Access

The kagent controller also provides a REST API:

```bash
# Forward controller port
kubectl -n kagent port-forward service/kagent-controller 8083:8083

# API endpoint
http://localhost:8083/api
```

---

## 📊 Current Deployment Status

### Pods Running
```
✅ kagent-ui-7f45b6cc55-kgflq (1/1 Running)
✅ kagent-controller-5d47dcff47-64ct9 (1/1 Running)
✅ kagent-kmcp-controller-manager (1/1 Running)
✅ kagent-tools (1/1 Running)
✅ kagent-grafana-mcp (1/1 Running)
✅ kagent-querydoc (1/1 Running)
```

### Services
```
✅ kagent-ui (ClusterIP 192.168.194.174:8080)
✅ kagent-controller (ClusterIP 192.168.194.156:8083)
```

### Installed via Helm
```
Release: kagent
Chart: kagent-0.7.7
Namespace: kagent
Status: deployed
```
---

## 🎮 Using the Kagent UI (short)

1. Start UI (recommended):
```bash
./agentstack/bin/agentctl ui --port 3000
open http://localhost:3000
```

2. Quick check:
```bash
kubectl -n kagent get pods
kubectl -n kagent get agents
```

3. In the UI: go to **Agents** → select agent → **Chat**.

---

## 🔐 Add LLM credentials (example: OpenAI)

```bash
kubectl -n kagent create secret generic openai-secret --from-literal=apiKey=YOUR_OPENAI_API_KEY
kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha1
kind: ModelConfig
metadata:
  name: openai-gpt4
  namespace: kagent
spec:
  provider: openAI
  model: gpt-4
  apiKeySecret:
    name: openai-secret
    key: apiKey
EOF
```

---

## 🤖 Create an agent (quick)

Save this as `my-agent.yaml` and apply:
```yaml
apiVersion: kagent.dev/v1alpha1
kind: Agent
metadata:
  name: my-first-agent
  namespace: kagent
spec:
  systemPrompt: "You are a helpful assistant about Kubernetes."
  modelConfig:
    name: openai-gpt4
  tools:
  - name: kagent-tools
```

```bash
kubectl apply -f my-agent.yaml
```

---

## 🔍 Troubleshooting (essentials)

- Check pods: `kubectl -n kagent get pods`
- UI logs: `kubectl -n kagent logs -l app.kubernetes.io/name=kagent-ui -f`
- Restart UI: `kubectl -n kagent rollout restart deployment/kagent-ui`
- Port-forward: `pkill -f "port-forward.*kagent-ui"` then re-run port-forward
- If a local port is busy, forward to a different local port (e.g., `8081:8080`)

---

## 📋 Useful Commands

### View All Kagent Resources
```bash
kubectl -n kagent get agents,modelconfigs,toolservers,memories
```

### Get Agents
```bash
kubectl -n kagent get agents
```

### View Agent Details
```bash
kubectl -n kagent describe agent <agent-name>
```

### View Logs
```bash
# UI logs
kubectl -n kagent logs -l app.kubernetes.io/name=kagent-ui -f

# Controller logs
kubectl -n kagent logs -l app.kubernetes.io/component=controller -f

# All kagent logs
kubectl -n kagent logs -l app.kubernetes.io/name=kagent -f
```

### Check Events
```bash
kubectl -n kagent get events --sort-by='.lastTimestamp'
```

### View All Pods
```bash
kubectl -n kagent get pods
```

---

## 🎓 Next Steps

### 1. Explore Pre-installed Agents
- Open UI at http://localhost:8080
- Click "Agents" tab
- Try chatting with k8s-agent
- Test helm-agent capabilities

### 2. Add Your LLM Credentials
- Create secret with API key
- Create ModelConfig resource
- Set as default for agents

### 3. Create Custom Agent
- Use UI or kubectl
- Define system prompt
- Select model config
- Add tools

### 4. Explore MCP Tools
- Browse available tools
- Add custom MCP servers
- Test tool capabilities

### 5. Monitor with Observability
- Enable OpenTelemetry tracing
- View agent traces in UI
- Monitor performance

---

## 📚 Documentation Resources

### Official Kagent Docs
- **Website**: https://kagent.dev
- **Getting Started**: https://kagent.dev/docs/kagent/getting-started/quickstart
- **Architecture**: https://kagent.dev/docs/kagent/concepts/architecture
- **Examples**: https://kagent.dev/docs/kagent/examples

### GitHub
- **Repository**: https://github.com/kagent-dev/kagent
- **Issues**: https://github.com/kagent-dev/kagent/issues
- **Discussions**: https://github.com/kagent-dev/kagent/discussions

### Community
- **Discord**: https://discord.gg/Fu3k65f2k3
- **Slack**: https://cloud-native.slack.com/archives/C08ETST0076

---

## 🔄 Updating Kagent

To update to the latest version:

```bash
# Update CRDs
helm upgrade kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds \
  --namespace kagent

# Update kagent
helm upgrade kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent \
  --namespace kagent \
  --set ui.enabled=true
```

---

## 🗑️ Uninstalling Kagent

To completely remove kagent:

```bash
# Uninstall kagent
helm uninstall kagent -n kagent

# Uninstall CRDs
helm uninstall kagent-crds -n kagent

# Delete namespace (optional)
kubectl delete namespace kagent
```

---

## ✅ Verification Checklist

- [x] Kagent installed via Helm
- [x] kagent-ui pod running (1/1 Ready)
- [x] kagent-controller pod running
- [x] UI service accessible on port 8080
- [x] 10 pre-installed agents deployed
- [x] 3 MCP tool servers running
- [x] Documentation complete

---

## 🎉 You're All Set!

Your official Kagent UI is deployed and ready to use!

**To access:**
```bash
kubectl -n kagent port-forward service/kagent-ui 8080:8080
```

**Then open:**
```
http://localhost:8080
```

**Enjoy managing your AI agents with the official Kagent UI!** 🚀

---

*Last Updated: 2025-12-16*  
*Kagent Version: 0.7.7*  
*Installation Method: Helm*  
*Status: ✅ Production Ready*
