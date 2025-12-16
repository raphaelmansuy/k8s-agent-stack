# Glossary

Key terms and concepts used in k8s-agent-stack.

---

## Core Concepts

| Term | Definition |
|------|------------|
| **Agent** | An AI-powered service that understands natural language, reasons, and takes actions. Built using frameworks like Google ADK, LangGraph, or CrewAI. |
| **A2A Protocol** | Agent-to-Agent communication protocol enabling agents to discover and communicate with each other. Part of [kagent](https://github.com/kagent-dev/kagent). |
| **Scale-to-Zero** | Automatically scaling down to zero pods when idle, saving resources. Knative Serving's key feature. |
| **Cold Start** | Latency when scaling from zero to handle the first request. Optimized by reducing image size or keeping warm pods. |

---

## Frameworks & Tools

| Term | Definition |
|------|------------|
| **[kagent](https://github.com/kagent-dev/kagent)** | CNCF Kubernetes-native framework for AI agent orchestration. Provides A2A protocol, MCP tools, and multi-framework support. |
| **[Knative](https://knative.dev/)** | Kubernetes-based platform for serverless workloads. Provides scale-to-zero, auto-scaling, and traffic management. |
| **[Google ADK](https://google.github.io/adk-docs/)** | Agent Developer Kit - Google's framework for building agentic applications with Gemini. |
| **[Contour](https://projectcontour.io/)** | Open-source ingress controller using Envoy for L7 routing. |
| **[Envoy](https://www.envoyproxy.io/)** | High-performance L7 proxy for load balancing and routing. |

---

## Knative Terms

| Term | Definition |
|------|------------|
| **Knative Service (ksvc)** | High-level abstraction managing routing, revisions, and scaling for your agent. Created with `kn service create`. |
| **Revision** | Immutable snapshot of code and configuration. Each deployment creates a new revision. |
| **Activator** | Knative component that buffers requests when scaling from zero and during scale-up. |
| **Autoscaler** | Knative component that monitors metrics and adjusts pod replicas. |
| **Queue-Proxy** | Sidecar container in each pod that reports metrics to the autoscaler. |

---

## Kubernetes Terms

| Term | Definition |
|------|------------|
| **Pod** | Smallest deployable unit in Kubernetes. Contains one or more containers running your agent. |
| **Deployment** | Kubernetes resource managing replica pods. Ensures desired number of pods are running. |
| **Service (K8s)** | Stable network endpoint for accessing pods. Provides service discovery and load balancing. |
| **Namespace** | Virtual cluster for organizing resources. Agents typically run in `kagent` namespace. |
| **ConfigMap** | Non-sensitive configuration stored in Kubernetes. |
| **Secret** | Sensitive data (API keys) stored securely in Kubernetes. |
| **Ingress** | HTTP(S) routing from outside the cluster. |

---

## Agent Development Terms

| Term | Definition |
|------|------------|
| **SSE** | Server-Sent Events - HTTP protocol for streaming data from server to client. |
| **Tool Calling** | Agent capability to execute predefined functions/APIs. |
| **Context Window** | Maximum text (tokens) an LLM can process in one request. |
| **Embeddings** | Numerical representations of text for semantic search. |
| **RAG** | Retrieval-Augmented Generation - Retrieve documents before generating responses. |
| **MCP** | Model Context Protocol - Standard for model-context integration (supported by kagent). |

---

## Autoscaling Metrics

| Term | Definition |
|------|------------|
| **scale-min** | Minimum pods to keep running. 0 = scale-to-zero, 1+ = warm pods. |
| **scale-max** | Maximum pods under load. |
| **concurrency-target** | Target concurrent requests per pod before scaling up. |
| **concurrency-limit** | Hard limit on concurrent requests per pod. |
| **scale-down-delay** | Wait time before scaling down idle pods (default: 60s). |

---

## Common Annotations

| Annotation | Purpose | Example |
|------------|---------|---------|
| `autoscaling.knative.dev/min-scale` | Minimum replicas | `"1"` |
| `autoscaling.knative.dev/max-scale` | Maximum replicas | `"10"` |
| `autoscaling.knative.dev/target` | Concurrency target | `"10"` |
| `autoscaling.knative.dev/class` | Autoscaler type | `"kpa"` or `"hpa"` |
| `autoscaling.knative.dev/metric` | Scaling metric | `"concurrency"` or `"rps"` |

---

## kagent CRDs

| Term | Definition |
|------|------------|
| **Agent** | Custom Resource defining an AI agent deployment. Can be BYO (bring your own) or built-in type. |
| **ModelConfig** | Custom Resource for LLM provider configuration (OpenAI, Anthropic, Gemini, etc.). |
| **ToolServer** | Custom Resource defining MCP tool servers that agents can use. |
| **Memory** | Custom Resource for agent memory/state persistence. |
| **BYO Agent** | "Bring Your Own" agent type - deploy custom container images as agents. |

---

## Status Icons

| Icon | Meaning |
|------|---------|
| ✅ | Production ready / Complete |
| 🚧 | In progress |
| 📋 | Planned |
| ⚠️ | Warning / Partial |
| ❌ | Not available |

---

## Related Docs

- [Architecture](architecture.md)
- [Getting Started](getting-started.md)
- [Troubleshooting](troubleshooting.md)

---

[← Back to Documentation Index](README.md) • [Architecture](architecture.md) • [Deployment Guide](deployment-guide.md) • [Main README](../README.md)
