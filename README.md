# k8s-agent-stack

[![kagent](https://img.shields.io/badge/kagent-CNCF-green)](https://github.com/kagent-dev/kagent)
[![Knative](https://img.shields.io/badge/Knative-1.20+-blue)](https://knative.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> Sovereign AI agent platform on Kubernetes. Deploy, orchestrate, and scale agents with agent-to-agent communication—on **your** infrastructure.

**Author**: [Raphaël MANSUY](https://www.linkedin.com/in/raphaelmansuy/)

![kagent demo](images/kagent-01.gif)

## Why k8s-agent-stack?

| Feature | k8s-agent-stack | Cloud Vendors |
|---------|-----------------|---------------|
| **Run anywhere** | ✅ Local, cloud, on-prem | ❌ Locked to vendor |
| **Your data** | ✅ Your infrastructure | ❌ Vendor servers |
| **A2A Protocol** | ✅ Open (kagent) | ❌ Proprietary |
| **Scale-to-zero** | ✅ Pay for what you use | ⚠️ Varies |
| **Multi-framework** | ✅ ADK, LangGraph, CrewAI | ❌ Limited |

## Quick Start (5 min)

```bash
# Prerequisites: kubectl, kn CLI, docker
# Install kn: brew install knative/client/kn

# 1. Clone & install
git clone https://github.com/raphaelmansuy/k8s-agent-stack.git
cd k8s-agent-stack
./knative_orbstack.sh

# 2. Deploy test agent
kn service create hello --image=gcr.io/knative-samples/helloworld-go --port=8080

# 3. Verify
curl $(kn service describe hello -o url)
```

📖 **[Full Installation Guide](docs/getting-started.md)** | **[Production Setup](docs/getting-started.md#production-deployment)**

## Architecture

```text
┌────────────────────────────────────────────────────────────┐
│  🤖 YOUR AGENTS (Google ADK │ LangGraph │ CrewAI │ Custom) 
├────────────────────────────────────────────────────────────┤
│  📡 kagent: A2A Protocol • Multi-Framework • Discovery    
├────────────────────────────────────────────────────────────┤
│  ⚡ Knative: Scale-to-Zero • Auto-Scaling • Traffic Mgmt  
├────────────────────────────────────────────────────────────┤
│  ☸️ Kubernetes: OrbStack │ GKE │ EKS │ AKS │ On-Prem      
└────────────────────────────────────────────────────────────┘
```

Implements a **5-layer agentic platform**: Runtime → Cognitive → Memory → Interface → Governance

📖 **[Architecture Deep Dive](docs/architecture.md)**

## Deploy Your First Agent

```bash
# Deploy the included Google ADK agent
cd kagent-adk-agent
kubectl apply -f kagent-deployment.yaml

# Test it
kubectl port-forward -n kagent svc/google-adk-agent 8080:8080 &
curl http://localhost:8080/health
```

📖 **[Deployment Guide](docs/deployment-guide.md)** | **[Build ADK Agents](docs/building-google-adk-agents-for-kagent.md)**

## Documentation

| Guide | Description |
|-------|-------------|
| **[Getting Started](docs/getting-started.md)** | Installation for local & production |
| **[Architecture](docs/architecture.md)** | 5-layer platform design |
| **[Deployment Guide](docs/deployment-guide.md)** | Deploy, update, traffic splitting |
| **[Troubleshooting](docs/troubleshooting.md)** | Common issues & solutions |
| **[Building ADK Agents](docs/building-google-adk-agents-for-kagent.md)** | Google ADK integration |
| **[Glossary](docs/glossary.md)** | Key terms explained |
| **[Tool Installation](docs/tool-installation.md)** | kubectl, kn, helm, docker |

## What's Included

- **[kagent](https://github.com/kagent-dev/kagent)** – CNCF agent orchestration with A2A protocol
- **[Knative Serving](https://knative.dev/docs/serving/)** – Serverless execution, auto-scaling
- **[Contour](https://projectcontour.io/) + Envoy** – L7 ingress & routing
- **Reference Agent** – [kagent-adk-agent/](kagent-adk-agent/) with Google ADK
- **40+ Makefile targets** – Build, test, deploy automation

## Repository Structure

```text
k8s-agent-stack/
├── knative_orbstack.sh          # Installer + diagnostics
├── kagent-adk-agent/            # Reference ADK agent
│   ├── app/agent.py             # Agent logic
│   ├── Dockerfile               # Container build
│   └── kagent-deployment.yaml   # K8s manifest
├── docs/                        # Documentation
├── examples/                    # Agent configurations
└── Makefile                     # Automation
```

## Roadmap

- [x] Knative + kagent + Google ADK integration
- [x] A2A protocol support
- [ ] LangGraph/CrewAI support (Q4 2025)
- [ ] Vector database integration (Q4 2025)
- [ ] MCP (Model Context Protocol) support (Q4 2025)
- [ ] Multi-cloud templates (Q1 2026)

## Contributing

We welcome contributions! See [CONTRIBUTORS.md](CONTRIBUTORS.md) for guidelines.

- 🐛 [Report bugs](https://github.com/raphaelmansuy/k8s-agent-stack/issues)
- 💡 [Request features](https://github.com/raphaelmansuy/k8s-agent-stack/issues)
- 📖 Improve documentation
- 🤖 Add agent examples

## License

Apache License 2.0 – see [LICENSE](LICENSE)

**Copyright © 2025 [Raphaël MANSUY](https://www.linkedin.com/in/raphaelmansuy/)**

---

<div align="center">

**k8s-agent-stack** — From zero to production AI agents in 5 minutes

[Get Started](docs/getting-started.md) • [Docs](docs/) • [Examples](examples/)

</div>
