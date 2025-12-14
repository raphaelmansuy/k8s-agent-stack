# Kagent Installation Summary

## Successfully Completed Installation

Date: December 13, 2025

### What Was Installed

1. **Kagent Core Components (v0.7.7)**
   - Controller (manages Kubernetes resources)
   - Engine (AI agent runtime)
   - UI (web interface at http://localhost:8080)
   - KMCP Controller (manages MCPServer resources)

2. **Pre-installed Tools**
   - Kagent Built-in Tool Server (http://kagent-tools.kagent:8084/mcp)
   - Grafana MCP Server (http://kagent-grafana-mcp.kagent:8000/mcp)
   - QueryDoc Tool

3. **Pre-installed Agents** (10 total)
   - argo-rollouts-agent
   - cilium-debug-agent
   - cilium-manager-agent
   - cilium-policy-agent
   - helm-agent
   - istio-agent
   - k8s-agent (tested and working!)
   - kgateway-agent
   - observability-agent
   - promql-agent

4. **Custom Resources Created**
   - ModelConfig: `gpt-4o-mini` (custom brain configuration)
   - Agent: `k8s-helper-agent` (custom Kubernetes helper)

5. **Kagent CLI**
   - Installed at: /usr/local/bin/kagent
   - Version: 0.7.7

### Installation Steps Executed

```bash
# 1. Install CRDs
helm install kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds \
  --namespace kagent --create-namespace

# 2. Install Kagent with OpenAI provider
helm install kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent \
  --namespace kagent \
  --set providers.default=openAI \
  --set providers.openAI.apiKey=$OPENAI_API_KEY

# 3. Install CLI
curl -sSL https://raw.githubusercontent.com/kagent-dev/kagent/refs/heads/main/scripts/get-kagent | bash
```

### Test Results

✅ Successfully invoked the k8s-agent with the query "List all pods in the kagent namespace"

The agent:
- Connected to the Kagent controller
- Used the k8s_get_resources tool from the MCP server
- Retrieved all 17 pods running in the kagent namespace
- Returned a formatted response

Sample invocation:
```bash
kagent invoke -a k8s-agent -t "List all pods in the kagent namespace" -n kagent
```

### Architecture Deployed

```
User → Kagent CLI → Controller API (port 8083)
                        ↓
                   Agent Pod (k8s-agent)
                        ↓
                   Tool Server (MCP)
                        ↓
                   Kubernetes API
```

### Custom Agent Created

File: `examples/k8s-helper-agent.yaml`

Features:
- Uses custom ModelConfig (gpt-4o-mini)
- Has access to 4 key k8s tools:
  - k8s_get_resources
  - k8s_describe_resource
  - k8s_get_pod_logs
  - k8s_get_events

### Access Points

- **UI Dashboard**: http://localhost:8080 (port-forward active)
- **Controller API**: http://localhost:8083 (port-forward active)
- **CLI**: `kagent` command available globally

### Available Tools in Tool Server

The kagent-tool-server provides 80+ tools including:
- Kubernetes operations (get, describe, create, delete, patch, etc.)
- Helm operations (install, upgrade, list, etc.)
- Istio management
- Cilium networking
- Argo Rollouts
- Prometheus queries
- Shell execution

### Next Steps

1. Explore the UI at http://localhost:8080
2. Try different agents with `kagent invoke`
3. Create custom agents for specific use cases
4. Add custom MCP servers with domain-specific tools
5. Explore A2A (Agent-to-Agent) interactions

### Useful Commands

```bash
# List all agents
kubectl get agents -n kagent

# Check agent status
kubectl get agent k8s-helper-agent -n kagent

# View agent logs
kubectl logs -n kagent -l app.kubernetes.io/name=k8s-agent

# Invoke an agent
kagent invoke -a k8s-agent -t "your question here" -n kagent

# Open dashboard
kagent dashboard
```

### Documentation References

- Official Docs: https://kagent.dev
- GitHub: https://github.com/kagent-dev/kagent
- Discord: https://discord.gg/Fu3k65f2k3

## Conclusion

Kagent has been successfully installed and tested. The platform is ready for:
- Building AI agents that interact with Kubernetes
- Automating cluster operations
- Providing conversational interfaces to infrastructure
- Creating custom agents for specific workflows

All components are running and operational! 🚀
