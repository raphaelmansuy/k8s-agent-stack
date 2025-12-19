# k8s-agent-stack Documentation

Complete documentation for building, deploying, and operating AI agents on Kubernetes.

📋 **[Documentation Cross-Reference Index](DOCUMENTATION_INDEX.md)** - Complete verification of all documents and links

## 📚 Guides

| Guide | Description |
|-------|-------------|
| **[Getting Started](getting-started.md)** | Installation for local development and production |
| **[Architecture](architecture.md)** | 5-layer agentic platform design (see [Deep Dives](architecture/)) |
| **[Deployment Guide](deployment-guide.md)** | Deploy, update, traffic splitting, canary releases |
| **[Quick Reference](quick-reference.md)** | Essential commands and common workflows |
| **[Troubleshooting](troubleshooting.md)** | Common issues and solutions |
| **[Tool Installation](tool-installation.md)** | kubectl, kn, helm, docker setup |
| **[Glossary](glossary.md)** | Key terms and concepts |
| **[Governance](governance.md)** | Project governance and decision-making |

## 🤖 Agent Development

| Guide | Description |
|-------|-------------|
| **[Building Google ADK Agents](building-google-adk-agents-for-kagent.md)** | Complete guide from zero to deployed agent |
| **[ADK Guide](adk-guide.md)** | Comprehensive ADK reference and cheatsheet |
| **[kagent A2A Architecture](kagent-adk-a2a-architecture.md)** | Agent-to-agent communication protocol |
| **[Tutorials](tutorial/README.md)** | Step-by-step tutorials for common tasks |

## 📋 Reference

| Document | Description |
|----------|-------------|
| **[Specifications](spec/README.md)** | Detailed design specifications for the platform |
| **[UI Access Guide](KAGENT_UI_ACCESS.md)** | Accessing and using the Kagent Web UI |

## 📁 Assets

The `images/` directory contains:
- Architecture diagrams
- kagent Web UI screenshots
- Agent response examples

## 🔗 External Resources

- **[kagent Documentation](https://kagent.dev/docs/)** - Official kagent docs
- **[Knative Serving](https://knative.dev/docs/serving/)** - Serverless platform
- **[Google ADK](https://google.github.io/adk-docs/)** - Agent Developer Kit
- **[Contour](https://projectcontour.io/docs/)** - Ingress controller

---

[← Back to Main README](../README.md) • [Examples](../examples/) • [Agent Implementation](../kagent-adk-agent/)
