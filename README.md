# k8s-agent-stack — Sovereign AI Agent Platform

[![kagent](https://img.shields.io/badge/kagent-powered-green)](https://github.com/kagent-dev/kagent)
[![Knative](https://img.shields.io/badge/Knative-1.12+-blue)](https://knative.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> Build, orchestrate, and scale AI agents on YOUR infrastructure. A sovereign platform with agent-to-agent communication, multi-framework support, and true vendor independence. From zero to production agents in 5 minutes.

**Author**: [Raphaël MANSUY](https://www.linkedin.com/in/raphaelmansuy/)

```
┌─────────────────────────────────────────────────────────────┐
│           🤖 YOUR AI AGENTS (Sovereign & Portable)          
│  Google ADK │ LangGraph │ CrewAI │ AutoGen │ Custom         │
├─────────────────────────────────────────────────────────────┤
│         📡 kagent: Agent Orchestration Platform             
│  • A2A Protocol (Agent-to-Agent Communication)              │
│  • Multi-Framework Support (ADK, LangGraph, CrewAI)         │
│  • Service Discovery & Lifecycle Management                 │
├─────────────────────────────────────────────────────────────┤
│         ⚡ Knative: Serverless Execution Layer              
│  • Scale-to-Zero (Cost Optimization)                        │
│  • Auto-Scaling (Handle Any Load)                           │
│  • Traffic Management (Canary, Blue-Green)                  │
├─────────────────────────────────────────────────────────────┤
│         ☸️  Run Anywhere: True Sovereignty                  
│  OrbStack │ GKE │ EKS │ AKS │ On-Prem │ Any K8s             │
└─────────────────────────────────────────────────────────────┘
```
![kagent-01](images/kagent-01.gif)

## 📚 Documentation Navigator

| Quick Links | Description |
|-------------|-------------|
| **[🎯 What is Sovereignty?](#-what-is-sovereignty-in-ai-agents)** | Why this matters |
| **[⚡ Quick Install](#-quick-install-5-minutes)** | Get started in 5 minutes |
| **[🏗️ Architecture](#-architecture-overview)** | Understand the stack |
| **[🚀 Deploy Your First Agent](#-deploy-your-first-agent)** | Step-by-step guide |
| **[📖 Complete Guides](docs/)** | Deep-dive documentation |
| **[🔍 Troubleshooting](#-troubleshooting-guide)** | Common issues & solutions |
| **[📘 Glossary](#-glossary)** | Key terms explained |

## 🎯 What is Sovereignty in AI Agents?

**Sovereignty means YOU control your AI agents, not a vendor.**

### The Problem with Cloud Agent Platforms

| Platform | Lock-In | Your Data | Portability | A2A Protocol |
|----------|---------|-----------|-------------|--------------|
| **Google Vertex AI** | ⚠️ Google Cloud only | ⚠️ Google's servers | ❌ None | ❌ Proprietary |
| **AWS Bedrock** | ⚠️ AWS only | ⚠️ AWS servers | ❌ None | ❌ Proprietary |
| **Azure AI Studio** | ⚠️ Azure only | ⚠️ Microsoft servers | ❌ None | ❌ Proprietary |
| **k8s-agent-stack** | ✅ Run anywhere | ✅ YOUR infrastructure | ✅ Full | ✅ Open (kagent) |

### What k8s-agent-stack Provides

```
┌──────────────────────────────────────────────────────────┐
│  YOUR SOVEREIGN AGENT PLATFORM                           │
├──────────────────────────────────────────────────────────┤
│  ✓ Own your infrastructure (local, cloud, on-prem)       │
│  ✓ Own your data (no vendor access)                      │
│  ✓ Own your agents (portable, framework-agnostic)        │
│  ✓ Own your communication (open A2A protocol)            │
│  ✓ Own your future (no migration costs, no surprises)    │
└──────────────────────────────────────────────────────────┘
```

**Powered by:**
- **[kagent](https://github.com/kagent-dev/kagent)**: Open-source agent orchestration with A2A protocol
- **[Knative](https://knative.dev/)**: CNCF serverless platform (run on ANY Kubernetes)
- **Your choice**: Local development, cloud deployment, or air-gapped environments

---

## Why k8s-agent-stack? True Sovereignty for AI Agents

### 🏆 Sovereignty & Control
- ✅ **Own Your Agents**: No vendor lock-in, no proprietary platforms, full control
- ✅ **Agent-to-Agent Communication**: Native [A2A protocol](https://github.com/kagent-dev/kagent) support via [kagent](https://github.com/kagent-dev/kagent)
- ✅ **Multi-Framework Freedom**: Google ADK, LangGraph, CrewAI, AutoGen, or your custom agents
- ✅ **Run Anywhere**: Local (OrbStack), cloud (GKE/EKS/AKS), on-prem, air-gapped environments

### ⚡ Production-Grade Agent Platform
- ✅ **5-Minute Setup**: From zero to production agents with one command
- ✅ **Scale-to-Zero**: Pay only for what you use (powered by [Knative Serving](https://knative.dev/docs/serving/))
- ✅ **Auto-Scaling**: Handle any load, from 0 to 1000+ concurrent agents
- ✅ **Battle-Tested**: Built on Kubernetes, Knative, and kagent best practices

### 🚀 Developer Experience
- ✅ **Agent Orchestration**: Lifecycle management, service discovery, health checks
- ✅ **Easy Updates**: Canary deployments, traffic splitting, zero-downtime updates
- ✅ **Observable**: Built-in monitoring, logging, and debugging tools
- ✅ **Extensible**: Plug in any agent framework or LLM provider


---

## ⚡ Quick Install (5 minutes)

### Prerequisites

```bash
# Verify you have these installed:
kubectl version --client    # Kubernetes CLI
kn version                  # Knative CLI (install: brew install knative/client/kn)
docker --version            # Container runtime

# Optional but recommended:
helm version               # Package manager (install: brew install helm)
```

### Option 1: Local Development (OrbStack/kind)

```bash
# 1. Clone repository
git clone https://github.com/your-org/k8s-agent-stack.git
cd k8s-agent-stack

# 2. Run installer (installs Knative + Contour + Envoy)
./knative_orbstack.sh

# 3. Verify installation
kubectl get pods -n knative-serving
kubectl get pods -n projectcontour

# 4. Deploy test agent
kn service create hello \
  --image=gcr.io/knative-samples/helloworld-go \
  --port=8080

# 5. Test it works
curl $(kn service describe hello -o url)
```

**Installation time**: ~3-5 minutes ⏱️

See **[Local Development Guide](knative-orbstack.md)** for detailed setup.

### Option 2: Production Kubernetes (GKE/EKS/AKS)

```bash
# 1. Ensure kubectl is configured
kubectl cluster-info

# 2. Run installer with production settings
./knative_orbstack.sh --install-metrics --production

# 3. Configure domain (required for production)
kubectl edit configmap config-domain -n knative-serving
# Set: example.com: ""

# 4. Install cert-manager for TLS
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

See **[Production Guide](knative.md)** for complete production setup.

### Verify Installation

```bash
# Check all components are running
./knative_orbstack.sh --debug

# Should show:
# ✓ Knative Serving: Ready
# ✓ Contour: Ready
# ✓ Envoy: Ready
# ✓ DNS: Configured
```

---

## 🏗️ Architecture Overview: The 5-Layer Agentic Platform

k8s-agent-stack implements a comprehensive 5-layer architecture for sovereign AI agents, based on the Agentic Platform Reference Architecture (2025) published by Raphaël MANSUY in December 2025.

![Agentic Platform Architecture](images/agentic.png)

### The 5 Layers Explained

| Layer | Purpose | Current Components | Roadmap Components | Status |
|-------|---------|-------------------|-------------------|---------|
| **5. GOVERNANCE** | Security, compliance, observability | • Basic monitoring<br>• kubectl logs | • Prometheus/Grafana<br>• RBAC policies<br>• Guardrails<br>• Audit logging<br>• Cost tracking | 🚧 In Progress |
| **4. INTERFACE** | Agent communication & interaction | • **kagent** (A2A protocol)<br>• **Google ADK**<br>• FastAPI endpoints<br>• SSE streaming | • **MCP** (Model Context Protocol)<br>• **HITL** (Human-in-the-Loop)<br>• **Agentic RAG**<br>• WebSocket support | ✅ Partial |
| **3. MEMORY** | Agent state & knowledge management | • Kubernetes ConfigMaps<br>• Kubernetes Secrets | • **Redis** (short-term)<br>• **Vector DB** (semantic)<br>• **Episodic memory**<br>• Knowledge graphs | 📋 Planned |
| **2. COGNITIVE** | Reasoning & decision-making | • **Google ADK**<br>• Gemini integration | • **LangGraph** (workflows)<br>• **CrewAI** (multi-agent)<br>• **AutoGen**<br>• Model routing (GPT-5, Claude, SLM)<br>• ReAct/Reflection patterns | 🚧 In Progress |
| **1. RUNTIME** | Execution & orchestration | • **Kubernetes**<br>• **Knative Serving**<br>• **Contour/Envoy**<br>• metrics-server | • Advanced scheduling<br>• GPU support<br>• Edge deployment<br>• Durable execution | ✅ Production |

> 🏗️ **Note**: This stack is under active development. We're following a bottom-up implementation approach, starting with a solid runtime foundation and progressively adding cognitive, memory, interface, and governance capabilities. See our [Roadmap](#🚧-roadmap) for planned features.

### Layer Details

#### Layer 1: RUNTIME - Durable Execution & Orchestration
**Status**: ✅ **Production Ready**

The foundation layer handles container orchestration, serverless execution, and infrastructure portability.

```
┌─────────────────────────────────────────────────────────┐
│  RUNTIME LAYER: Execution Infrastructure                │
├─────────────────────────────────────────────────────────┤
│  ✓ Kubernetes: Container orchestration                  │
│  ✓ Knative Serving: Scale-to-zero, auto-scaling         │
│  ✓ Contour/Envoy: L7 routing, load balancing            │
│  ✓ metrics-server: Resource monitoring                  │
└─────────────────────────────────────────────────────────┘
```

**Key Features:**
- Serverless execution with scale-to-zero
- Auto-scaling from 0 to 1000+ concurrent agents
- Traffic management (canary, blue-green deployments)
- Multi-cloud portability (GKE, EKS, AKS, on-prem)

#### Layer 2: COGNITIVE - Reasoning & Model Routing
**Status**: 🚧 **In Progress** (Google ADK available, LangGraph/CrewAI coming soon)

The cognitive layer handles agent reasoning, decision-making, and LLM interactions.

```
┌─────────────────────────────────────────────────────────┐
│  COGNITIVE LAYER: AI Reasoning Engine                   │
├─────────────────────────────────────────────────────────┤
│  ✓ Google ADK: Structured agent development            
│  ✓ Gemini integration: LLM backbone                    
│  🚧 LangGraph: Complex workflows (Q4 2025)             
│  🚧 CrewAI: Multi-agent orchestration (Q4 2025)        
│  📋 Model routing: GPT-5, Claude, SLM (Q4 2025)        
│  📋 ReAct/Reflection patterns (Q4 2025)                
└─────────────────────────────────────────────────────────┘
```

**Current Capabilities:**
- Google ADK agent framework
- Tool calling and function execution
- Streaming responses via SSE

**Coming Soon:**
- LangGraph for complex agent workflows
- CrewAI for multi-agent collaboration
- Intelligent model routing across providers
- Advanced reasoning patterns (ReAct, Reflection, Chain-of-Thought)

#### Layer 3: MEMORY - State & Knowledge Management
**Status**: 📋 **Planned** (Q4 2025)

The memory layer will provide agents with short-term, episodic, and semantic memory capabilities.

```
┌─────────────────────────────────────────────────────────┐
│  MEMORY LAYER: Agent Knowledge & State                  │
├─────────────────────────────────────────────────────────┤
│  📋 Short-term: Redis, session management               
│  📋 Episodic: Conversation history, event logs          
│  📋 Semantic: Vector databases (Pinecone, Weaviate)    
│  📋 Knowledge graphs: Entity relationships             
└─────────────────────────────────────────────────────────┘
```

**Planned Features:**
- Redis for fast short-term memory and session state
- Vector databases for semantic search and RAG
- Episodic memory for conversation context
- Knowledge graph integration

#### Layer 4: INTERFACE - Communication & Interaction
**Status**: ✅ **Partial** (A2A available, MCP/HITL/RAG coming soon)

The interface layer enables agent-to-agent communication, human interaction, and external integrations.

```
┌─────────────────────────────────────────────────────────┐
│  INTERFACE LAYER: Communication Protocols               │
├─────────────────────────────────────────────────────────┤
│  ✓ A2A Protocol: Agent-to-agent messaging (kagent)      │
│  ✓ Google ADK: Structured I/O                           │
│  ✓ REST/SSE: HTTP endpoints, streaming                  │
│  🚧 MCP: Model Context Protocol (Q1 2025)               
│  📋 HITL: Human-in-the-Loop workflows (Q1 2025)         
│  📋 Agentic RAG: Retrieval-augmented generation          
└─────────────────────────────────────────────────────────┘
```

**Current Capabilities:**
- A2A protocol via kagent for agent-to-agent communication
- RESTful APIs with FastAPI
- Server-Sent Events (SSE) for streaming
- Google ADK structured input/output

**Coming Soon:**
- Model Context Protocol (MCP) support
- Human-in-the-Loop (HITL) workflows
- Agentic RAG with vector search
- WebSocket support for real-time bidirectional communication

#### Layer 5: GOVERNANCE - Security & Observability
**Status**: 🚧 **In Progress** (Basic monitoring available)

The governance layer ensures security, compliance, cost control, and system observability.

```
┌─────────────────────────────────────────────────────────┐
│  GOVERNANCE LAYER: Security & Compliance                │
├─────────────────────────────────────────────────────────┤
│  ✓ Basic monitoring: kubectl logs, events              
│  🚧 Prometheus/Grafana: Metrics & dashboards           
│  📋 RBAC: Role-based access control                    
│  📋 Guardrails: Policy enforcement, safety checks      
│  📋 Audit logging: Compliance tracking                 
│  📋 Cost tracking: LLM usage & infrastructure costs    
└─────────────────────────────────────────────────────────┘
```

**Current Capabilities:**
- Basic Kubernetes logging and events
- Resource monitoring via metrics-server
- ConfigMaps and Secrets management

**Coming Soon:**
- Prometheus and Grafana for comprehensive metrics
- RBAC policies for fine-grained access control
- Guardrails for agent safety and policy enforcement
- Audit logging for compliance
- Cost tracking dashboards for LLM usage

### System Interaction Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    5. GOVERNANCE (Monitoring)                   │
│                    Observability • RBAC • Guardrails            │
└───────────────────┬─────────────────────────────────────────────┘
                    │ (Monitors all layers)
                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Internet / Users                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                   ┌─────────▼─────────┐
                   │  4. INTERFACE     │
                   │  A2A • MCP • HITL │  (kagent, Google ADK)
                   └─────────┬─────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
    ┌────▼─────┐      ┌─────▼──────┐     ┌─────▼──────┐
    │  Agent 1 │      │  Agent 2   │     │  Agent N   │
    │          │      │            │     │            │
    │  ┌───────┴──────┴────────┬───┴─────┴───────┐    │
    │  │  2. COGNITIVE         │                  │   │
    │  │  Reasoning • Models   │ ◄─────┐          │   │
    │  └───────────────────────┘       │          │   │
    │                                  │          │   │
    │  ┌───────────────────────────────▼─────┐    │   │
    │  │  3. MEMORY                          │    │   │
    │  │  Short-term • Episodic • Semantic   │    │   │
    │  └─────────────────────────────────────┘    │   │
    └────┬─────┘      └─────┬──────┘     └─────┬──────┘
         │                  │                   │
    ┌────▼──────────────────▼───────────────────▼─────┐
    │         1. RUNTIME (Knative Serving)            │
    │  Scale-to-Zero • Auto-scaling • Orchestration   │
    └────────────────────┬────────────────────────────┘
                         │
    ┌────────────────────▼────────────────────────────┐
    │     Kubernetes • Contour/Envoy • Infrastructure │
    └─────────────────────────────────────────────────┘
```

📖 **Read more**: [Architecture Deep Dive](docs/kagent-adk-a2a-architecture.md)


---

## 🚀 Deploy Your First Agent

### Method 1: Pre-built Google ADK Agent with kagent

Deploy the included Google ADK agent with A2A protocol and streaming support:

```bash
# 1. Navigate to agent directory
cd kagent-adk-agent

# 2. Deploy to Kubernetes
kubectl apply -f kagent-deployment.yaml

# 3. Wait for ready (30-60 seconds)
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=google-adk-agent \
  -n kagent --timeout=120s

# 4. Test the agent
kubectl port-forward -n kagent svc/google-adk-agent 8080:8080 &
curl http://localhost:8080/health
```

📖 **Full guide**: [Building Google ADK Agents](docs/building-google-adk-agents-for-kagent.md)

### Method 2: Deploy Custom Agent with Knative

```bash
# 1. Create a Knative Service
kn service create my-agent \
  --image=gcr.io/your-project/my-agent:v1 \
  --port=8080 \
  --env OPENAI_API_KEY=sk-... \
  --env MODEL=gpt-4 \
  --scale-min=0 \
  --scale-max=10 \
  --concurrency-target=10

# 2. Get the URL
AGENT_URL=$(kn service describe my-agent -o url)
echo "Agent URL: $AGENT_URL"

# 3. Test the agent
curl -X POST $AGENT_URL/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is the weather in San Francisco?",
    "session_id": "test-session-123"
  }'
```

### Agent Development Workflow

```
┌──────────────┐
│  1. Develop  │  Write agent code locally
│  Locally     │  Test with docker run
└──────┬───────┘
       │
┌──────▼───────┐
│  2. Build    │  docker build -t agent:v1
│  Container   │  
└──────┬───────┘
       │
┌──────▼───────┐
│  3. Deploy   │  kn service create agent --image=agent:v1
│  to K8s      │  or kubectl apply -f deployment.yaml
└──────┬───────┘
       │
┌──────▼───────┐
│  4. Test     │  curl agent-url/endpoint
│  & Monitor   │  kubectl logs, watch pods scale
└──────┬───────┘
       │
┌──────▼───────┐
│  5. Update   │  kn service update agent --image=agent:v2
│  Version     │  Traffic split for canary
└──────────────┘
```

---

## 📦 What's Included

### Agent Orchestration Layer
- ✅ **[kagent](https://github.com/kagent-dev/kagent)** - Agent orchestration with A2A protocol, service discovery, and lifecycle management
- ✅ **A2A Protocol** - Open standard for agent-to-agent communication
- ✅ **Multi-Framework Support** - Google ADK, LangGraph, CrewAI, AutoGen, custom agents
- ✅ **Example Agents** - Ready-to-deploy Google ADK agent with streaming and A2A support

### Serverless Execution Layer
- ✅ **[Knative Serving](https://knative.dev/docs/serving/)** v1.12+ - Scale-to-zero, auto-scaling, traffic management
- ✅ **[Contour](https://projectcontour.io/)** + **Envoy** - L7 ingress with intelligent routing
- ✅ **metrics-server** (optional) - Resource metrics for autoscaling
- ✅ **MetalLB** (optional) - Load balancer for bare-metal clusters

### Developer Tools & Documentation
- ✅ **kn CLI** - Knative command-line interface
- ✅ **Debug scripts** - Diagnostics and troubleshooting helpers
- ✅ **Test suites** - Integration and unit tests
- ✅ **Makefile** - 40+ automation targets for common operations

### Complete Documentation
- 📖 [kagent Architecture](docs/kagent-adk-a2a-architecture.md) - A2A protocol and agent orchestration
- 📖 [Building ADK Agents](docs/building-google-adk-agents-for-kagent.md) - Google ADK integration guide
- 📖 [kagent Installation](docs/KAGENT_INSTALLATION_SUMMARY.md) - Agent platform setup
- 📖 [Local Development](knative-orbstack.md) - OrbStack/kind setup
- 📖 [Production Deployment](knative.md) - GKE/EKS/AKS deployment

### Example Agents
- 🤖 **Google ADK Agent** - [kagent-adk-agent/](kagent-adk-agent/) - Full-featured reference
- 🤖 **K8s Helper Agent** - [examples/k8s-helper-agent.yaml](examples/k8s-helper-agent.yaml) - Natural language K8s interface

---

## 🗂️ Repository Structure

```
k8s-agent-stack/
├── 📄 README.md                      # ← You are here
├── ⚙️  knative_orbstack.sh           # Main installer + diagnostics
├── 📖 knative-orbstack.md            # Local dev quickstart
├── 📖 knative.md                     # Production deployment guide
│
├── 🤖 kagent-adk-agent/              # Reference ADK agent implementation
│   ├── Dockerfile                    # Multi-stage container build
│   ├── kagent-deployment.yaml        # K8s deployment manifest
│   ├── pyproject.toml                # Python dependencies
│   ├── app/
│   │   ├── agent.py                  # Agent logic (Google ADK)
│   │   ├── fast_api_app.py           # FastAPI server + SSE
│   │   └── app_utils/                # Telemetry & utilities
│   └── tests/
│       ├── integration/              # E2E tests
│       └── unit/                     # Unit tests
│
├── 📚 docs/                          # Deep-dive documentation
│   ├── kagent-adk-a2a-architecture.md        # A2A protocol & architecture
│   ├── building-google-adk-agents-for-kagent.md  # ADK development guide
│   ├── KAGENT_INSTALLATION_SUMMARY.md        # kagent setup
│   └── IMPLEMENTATION-COMPLETE.md            # Implementation status
│
├── 📋 examples/                      # Example configurations
│   ├── k8s-helper-agent.yaml         # Kubernetes assistant agent
│   └── model-config.yaml             # LLM model configurations
│
└── 📝 logs/                          # Development logs & notes
```

---

## 🛠️ Common Workflows

### 1. Deploy an Agent

```bash
# Option A: Using kagent manifest
kubectl apply -f kagent-adk-agent/kagent-deployment.yaml

# Option B: Using Knative Service directly
kn service create my-agent \
  --image=gcr.io/your-project/agent:latest \
  --port=8080 \
  --env GOOGLE_API_KEY=your-key \
  --scale-min=0 \
  --scale-max=5

# Option C: From local Docker image
docker build -t my-agent:local .
kn service create my-agent --image=my-agent:local --port=8080
```

### 2. Update an Agent

```bash
# Update image version
kn service update my-agent --image=gcr.io/your-project/agent:v2

# Update environment variables
kn service update my-agent \
  --env MODEL=gpt-4-turbo \
  --env MAX_TOKENS=2000

# Update scaling parameters
kn service update my-agent \
  --scale-min=1 \
  --scale-max=10 \
  --concurrency-target=20
```

### 3. Traffic Splitting (Canary Deployment)

```bash
# Deploy new version
kn service update my-agent --image=gcr.io/your-project/agent:v2

# Split traffic: 90% v1, 10% v2 (canary)
kn service update my-agent \
  --traffic my-agent-v1=90,@latest=10

# Gradually increase v2 traffic
kn service update my-agent \
  --traffic my-agent-v1=50,@latest=50

# Full rollout to v2
kn service update my-agent \
  --traffic @latest=100
```

```
Traffic Split Visualization:
  
  Users
    │
    ▼
┌─────────────┐
│   Envoy     │
└──────┬──────┘
       │
       ├─────90%─────▶ Agent v1 (stable)
       │
       └─────10%─────▶ Agent v2 (canary)
```

### 4. Monitor and Debug

```bash
# Check agent status
kubectl get ksvc -n kagent
kn service list

# View agent logs
kubectl logs -n kagent deployment/google-adk-agent --tail=50 --follow

# Check autoscaler decisions
kubectl logs -n knative-serving deploy/autoscaler --tail=30

# Port-forward for local testing
kubectl port-forward -n kagent svc/google-adk-agent 8080:8080

# Watch pods scale in real-time
watch 'kubectl get pods -n kagent'
```

### 5. Local Development Loop

```bash
# 1. Make code changes
vim kagent-adk-agent/app/agent.py

# 2. Build locally
cd kagent-adk-agent
docker build -t my-agent:dev .

# 3. Deploy to local cluster
kn service create my-agent --image=my-agent:dev --port=8080

# 4. Test
kubectl port-forward svc/my-agent 8080:8080 &
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "test"}'

# 5. Check logs for issues
kubectl logs -l serving.knative.dev/service=my-agent --tail=20

# 6. Iterate: repeat from step 1
```

---

## 🔧 Tool Installation Guide

### Install kn CLI (Knative)

```bash
# macOS
brew install knative/client/kn

# Linux
curl -L https://github.com/knative/client/releases/download/knative-v1.12.0/kn-linux-amd64 \
  -o /usr/local/bin/kn
chmod +x /usr/local/bin/kn

# Verify
kn version
```

**Official docs**: [Knative CLI](https://knative.dev/docs/client/install-kn/)

### Install kubectl

```bash
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Verify
kubectl version --client
```

**Official docs**: [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Install Helm (Optional)

```bash
# macOS
brew install helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify
helm version
```

**Official docs**: [Helm](https://helm.sh/docs/intro/install/)

### Install Docker/Podman

```bash
# macOS - Docker Desktop
brew install --cask docker

# macOS - OrbStack (recommended for M1/M2)
brew install --cask orbstack

# Linux - Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Verify
docker --version
```

**Official docs**: 
- [Docker](https://docs.docker.com/get-docker/)
- [OrbStack](https://orbstack.dev/)

---

## 🔍 Troubleshooting Guide

### Quick Diagnostics

```bash
# Run full system check
./knative_orbstack.sh --debug

# Output will show:
# ✓ Knative Serving: Ready
# ✓ Contour: Ready  
# ✓ Envoy: Ready
# ✓ DNS: Configured
# ✗ Issue detected: [description]
```

### Common Issues & Solutions

| Issue | Symptoms | Solution | Command |
|-------|----------|----------|---------|
| **Agent not responding** | HTTP 503 errors | Check pod status | `kubectl get pods -n kagent` |
| **Cold start too slow** | First request takes >10s | Reduce image size or set min replicas | `kn service update my-agent --scale-min=1` |
| **Ingress not working** | Can't reach agent URL | Check Envoy external IP | `kubectl get svc envoy -n projectcontour` |
| **Scaling not working** | Pods don't scale | Check autoscaler logs | `kubectl logs -n knative-serving deploy/autoscaler` |
| **Agent crashes** | CrashLoopBackOff | Check previous pod logs | `kubectl logs -n kagent deploy/my-agent --previous` |
| **Image pull errors** | ImagePullBackOff | Verify image exists | `docker pull <image>` |
| **DNS resolution fails** | nslookup errors | Check CoreDNS | `kubectl get pods -n kube-system -l k8s-app=kube-dns` |

### Debug Workflow

```
┌─────────────────────────────────────────────────────────┐
│  Problem: Agent not responding                          │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────▼────────────┐
        │ 1. Check Service Status │
        │  kn service list        │
        └────────────┬────────────┘
                     │
        ┌────────────▼────────────┐
        │ 2. Check Pods           │
        │  kubectl get pods       │
        └────────────┬────────────┘
                     │
        ┌────────────▼────────────┐
        │ 3. Check Logs           │
        │  kubectl logs <pod>     │
        └────────────┬────────────┘
                     │
        ┌────────────▼────────────┐
        │ 4. Check Events         │
        │  kubectl describe pod   │
        └────────────┬────────────┘
                     │
        ┌────────────▼────────────┐
        │ 5. Fix & Redeploy       │
        │  kn service update      │
        └─────────────────────────┘
```

### Detailed Troubleshooting Commands

```bash
# 1. Check overall service health
kn service list
kubectl get ksvc -n kagent

# 2. Inspect specific service
kn service describe my-agent
kubectl describe ksvc my-agent -n kagent

# 3. Check pod status
kubectl get pods -n kagent
kubectl describe pod <pod-name> -n kagent

# 4. View logs
kubectl logs -n kagent <pod-name> --tail=100
kubectl logs -n kagent <pod-name> --previous  # Previous crash

# 5. Check Knative system components
kubectl get pods -n knative-serving
kubectl logs -n knative-serving deploy/controller --tail=50
kubectl logs -n knative-serving deploy/autoscaler --tail=50
kubectl logs -n knative-serving deploy/activator --tail=50

# 6. Check ingress
kubectl get svc -n projectcontour
kubectl logs -n projectcontour deploy/contour --tail=50

# 7. Test connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- sh
# Inside pod:
curl http://my-agent.kagent.svc.cluster.local

# 8. Check resource usage
kubectl top pods -n kagent
kubectl top nodes
```

### Logs and Observability

```bash
# Stream logs in real-time
kubectl logs -f -n kagent deployment/google-adk-agent

# Get logs from all pods with label
kubectl logs -n kagent -l app.kubernetes.io/name=google-adk-agent --tail=50

# Export logs to file
kubectl logs -n kagent deployment/google-adk-agent --tail=500 > agent-logs.txt

# Check events in namespace
kubectl get events -n kagent --sort-by='.lastTimestamp'
```

### Performance Optimization

```bash
# Check cold start time
time curl $(kn service describe my-agent -o url)/health

# Monitor scaling behavior
watch 'kubectl get pods -n kagent -o wide'

# Check resource utilization
kubectl top pod -n kagent

# Adjust autoscaling
kn service update my-agent \
  --scale-min=1 \        # Keep 1 warm
  --scale-max=10 \       # Max 10 replicas
  --concurrency-target=10 \  # 10 requests per pod
  --concurrency-limit=50     # Max 50 requests per pod
```

---

## 📘 Glossary

### Core Concepts

| Term | Definition |
|------|------------|
| **Agent** | An AI-powered service that can understand natural language, reason, and take actions. Built using frameworks like Google ADK, LangGraph, or CrewAI. |
| **[Knative](https://knative.dev/)** | Kubernetes-based platform for deploying and managing serverless workloads. Provides scale-to-zero, auto-scaling, and traffic management. |
| **[kagent](https://github.com/kagent-dev/kagent)** | Agent orchestration framework that provides A2A protocol support and Google ADK integration for Kubernetes. |
| **Scale-to-Zero** | Ability to automatically scale down to zero pods when there's no traffic, saving resources. Knative Serving's key feature. |
| **Cold Start** | The latency incurred when scaling from zero to handle the first request. Optimized by reducing image size and using warm pods. |
| **Revision** | An immutable snapshot of your agent's code and configuration. Each deployment creates a new revision. |
| **Service (Knative)** | A high-level abstraction that manages routing, revisions, and scaling for your agent. Created with `kn service create`. |
| **A2A Protocol** | Agent-to-Agent communication protocol enabling agents to discover and communicate with each other. |
| **Google ADK** | [Agent Developer Kit](https://cloud.google.com/products/agent-developer-toolkit) - Google's framework for building agentic applications with Gemini. |

### Kubernetes Terms

| Term | Definition |
|------|------------|
| **Pod** | Smallest deployable unit in Kubernetes. Contains one or more containers running your agent. |
| **Deployment** | Kubernetes resource that manages a set of replica pods. Ensures desired number of pods are running. |
| **Service (K8s)** | Stable network endpoint for accessing pods. Provides service discovery and load balancing. |
| **Namespace** | Virtual cluster for organizing resources. Agents typically run in `kagent` or `default` namespace. |
| **ConfigMap** | Configuration data stored in Kubernetes. Used for non-sensitive configuration like model parameters. |
| **Secret** | Sensitive data like API keys stored securely in Kubernetes. Base64 encoded. |
| **Ingress** | HTTP(S) routing to services from outside the cluster. We use Contour/Envoy for this. |

### Knative-Specific Terms

| Term | Definition |
|------|------------|
| **ksvc** | Short for Knative Service. The CLI abbreviation used in `kubectl get ksvc`. |
| **Activator** | Knative component that buffers requests when scaling from zero and during scale-up. |
| **Autoscaler** | Knative component that monitors metrics and adjusts pod replicas. |
| **Queue-Proxy** | Sidecar container in each pod that reports metrics to the autoscaler. |
| **Contour** | [Open-source ingress controller](https://projectcontour.io/) that routes traffic to Knative services using Envoy. |
| **Envoy** | High-performance L7 proxy used by Contour for load balancing and routing. |
| **Concurrency** | Number of simultaneous requests a single pod can handle. Used for autoscaling decisions. |
| **Traffic Split** | Percentage-based routing between revisions. Used for canary deployments and A/B testing. |

### Agent Development Terms

| Term | Definition |
|------|------------|
| **Streaming** | Sending responses incrementally as they're generated. Uses Server-Sent Events (SSE) in HTTP. |
| **SSE** | Server-Sent Events - HTTP protocol for streaming data from server to client. Alternative to WebSockets. |
| **Tool Calling** | Agent capability to execute predefined functions/APIs to interact with external systems. |
| **Context Window** | Maximum amount of text (measured in tokens) an LLM can process in a single request. |
| **Embeddings** | Numerical representations of text used for semantic search and retrieval. |
| **RAG** | Retrieval-Augmented Generation - Pattern where agents retrieve relevant documents before generating responses. |
| **Prompt Engineering** | Crafting input instructions to optimize LLM behavior and output quality. |
| **Multi-Agent** | System where multiple specialized agents collaborate to solve complex tasks. |

### Autoscaling Metrics

| Term | Definition |
|------|------------|
| **scale-min** | Minimum number of pods to keep running. Set to 0 for scale-to-zero, 1+ for warm pods. |
| **scale-max** | Maximum number of pods to create under load. |
| **concurrency-target** | Target number of concurrent requests per pod before scaling up. |
| **concurrency-limit** | Hard limit on concurrent requests per pod. Requests above this are queued. |
| **scale-down-delay** | Time to wait before scaling down idle pods. Default is 60 seconds. |

### Common Annotations

| Annotation | Purpose | Example |
|------------|---------|---------|
| `autoscaling.knative.dev/min-scale` | Set minimum replicas | `"1"` (keep 1 warm) |
| `autoscaling.knative.dev/max-scale` | Set maximum replicas | `"10"` |
| `autoscaling.knative.dev/target` | Concurrency target | `"10"` (10 req/pod) |
| `autoscaling.knative.dev/class` | Autoscaler type | `"kpa"` or `"hpa"` |
| `autoscaling.knative.dev/metric` | Scaling metric | `"concurrency"` or `"rps"` |

---

## 🎯 Best Practices

### Agent Development
- ✅ **Use small base images**: `python:3.12-slim` or `distroless` for faster cold starts
- ✅ **Implement health checks**: `/health` (liveness) and `/readiness` endpoints
- ✅ **Stream long responses**: Use SSE for better UX on long-running tasks
- ✅ **Handle errors gracefully**: Return proper HTTP status codes and error messages
- ✅ **Log structured data**: Use JSON logging for better observability
- ✅ **Make agents stateless**: Store session data externally (Redis, database)
- ✅ **Set timeouts**: Configure reasonable request timeouts (30-60s for AI tasks)

### Deployment
- ✅ **Start with scale-min=0**: Test scale-to-zero works, then add warm pods if needed
- ✅ **Set resource limits**: Prevent agents from consuming too many resources
  ```yaml
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 2Gi
  ```
- ✅ **Use traffic splitting**: Test new versions with 10% traffic before full rollout
- ✅ **Tag images properly**: Use semantic versioning (`v1.0.0`) not `latest`
- ✅ **Store secrets securely**: Use K8s Secrets or External Secrets Operator

### Production
- ✅ **Monitor everything**: Set up Prometheus + Grafana for metrics
- ✅ **Set up alerts**: Alert on high error rates, slow responses, pod crashes
- ✅ **Enable TLS**: Use cert-manager for automatic certificate management
- ✅ **Implement rate limiting**: Protect against abuse and control LLM costs
- ✅ **Back up configurations**: Store manifests in Git for disaster recovery
- ✅ **Test failure scenarios**: Verify agent behavior when LLM API is down
- ✅ **Document runbooks**: Create troubleshooting guides for common issues

📖 **See also**: [Production Checklist](#production-checklist) for complete list

---

## � Production Checklist

Use this checklist when deploying to production environments.

### Infrastructure Setup
- [ ] Install metrics-server for pod autoscaling: `kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml`
- [ ] Deploy Prometheus + Grafana for monitoring
- [ ] Set up cert-manager for TLS: `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml`
- [ ] Configure custom domain in Knative: `kubectl edit configmap config-domain -n knative-serving`
- [ ] Set up DomainMapping for agent URLs
- [ ] Implement network policies for namespace isolation
- [ ] Configure backup strategy for etcd and persistent volumes

### Agent Configuration
- [ ] Define resource requests and limits:
  ```yaml
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 2Gi
  ```
- [ ] Configure autoscaling parameters:
  - `scale-min`: 0 (scale-to-zero) or 1+ (warm pods)
  - `scale-max`: Based on expected load
  - `concurrency-target`: 10-50 concurrent requests per pod
- [ ] Implement health checks:
  - `/health` (liveness probe)
  - `/readiness` (readiness probe)
- [ ] Set up structured logging (JSON format)
- [ ] Configure distributed tracing (Jaeger/Zipkin)
- [ ] Implement secrets management:
  - Use Kubernetes Secrets for API keys
  - Consider External Secrets Operator or Sealed Secrets
  - Rotate secrets regularly

### Observability
- [ ] Deploy monitoring stack:
  ```bash
  # Prometheus
  kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
  
  # Grafana
  kubectl apply -f grafana-deployment.yaml
  ```
- [ ] Set up log aggregation (ELK, Loki, or cloud provider)
- [ ] Configure alerting rules:
  - Agent pod crashes (>3 restarts in 5 minutes)
  - High error rate (>5% 5xx errors)
  - Slow response time (p95 > 30s)
  - Scale-up failures
  - LLM API rate limit errors
- [ ] Create dashboards for:
  - Request rate and latency
  - Pod scaling metrics
  - Error rates by agent
  - LLM API usage and costs
  - Cold start duration

### Security
- [ ] Enable TLS for all agent endpoints
- [ ] Rotate API keys quarterly
- [ ] Implement authentication:
  - API keys for service-to-service
  - OAuth2/OIDC for user-facing agents
- [ ] Set up RBAC for Kubernetes access:
  ```bash
  kubectl create role agent-reader --verb=get,list,watch --resource=pods,services
  kubectl create rolebinding agent-reader-binding --role=agent-reader --serviceaccount=kagent:default
  ```
- [ ] Scan container images for vulnerabilities:
  ```bash
  docker scan my-agent:v1.0.0
  # or use Trivy, Snyk, etc.
  ```
- [ ] Implement rate limiting per agent/user
- [ ] Set up network policies to restrict pod-to-pod communication
- [ ] Enable Pod Security Standards (restricted profile)

### Testing & Validation
- [ ] Run integration tests:
  ```bash
  pytest tests/integration/ -v --cov=app
  ```
- [ ] Perform load testing:
  ```bash
  hey -z 60s -c 100 -m POST \
    -H "Content-Type: application/json" \
    -d '{"message":"test"}' \
    https://my-agent.example.com/chat
  ```
- [ ] Validate scale-to-zero and scale-up behavior
- [ ] Test failure scenarios:
  - LLM API down
  - Database connection lost
  - High latency
  - Out of memory
- [ ] Verify backup and restore procedures
- [ ] Test disaster recovery plan

### Cost Optimization
- [ ] Set appropriate scale-min (0 for dev, 1+ for prod critical agents)
- [ ] Monitor LLM API costs per agent
- [ ] Implement caching for repeated queries
- [ ] Use smaller models where appropriate
- [ ] Set request timeouts to prevent runaway costs
- [ ] Configure resource limits to prevent overconsumption

### Documentation
- [ ] Create runbooks for common operations
- [ ] Document agent APIs (OpenAPI/Swagger)
- [ ] Write troubleshooting guides
- [ ] Document incident response procedures
- [ ] Maintain changelog for agent versions
- [ ] Create architecture diagrams

### Compliance & Governance
- [ ] Data retention policies configured
- [ ] PII handling procedures documented
- [ ] Audit logging enabled
- [ ] Compliance requirements met (GDPR, SOC2, etc.)
- [ ] Terms of service and usage policies defined

---

## �🚧 Roadmap

### ✅ Phase 1: Foundation (Complete)
- [x] Knative Serving installation script
- [x] kagent integration
- [x] Google ADK agent example
- [x] A2A protocol support
- [x] Documentation and guides

### 🚀 Phase 2: Q1 2025 (In Progress)
- [ ] LangGraph deployment support
- [ ] Vector database integration (Pinecone, Weaviate)
- [ ] Enhanced monitoring dashboards
- [ ] Agent catalog/marketplace
- [ ] Multi-agent orchestration examples

### 📅 Phase 3: Q2 2025
- [ ] CrewAI framework support
- [ ] AutoGen compatibility
- [ ] LLM gateway with cost optimization
- [ ] Advanced traffic management patterns
- [ ] Agent performance profiling tools

### 🔮 Future
- [ ] Multi-cloud deployment templates (GKE, EKS, AKS)
- [ ] Agent versioning and rollback strategies
- [ ] Federated agent networks
- [ ] Agent marketplace with pre-built templates
- [ ] Visual agent builder/workflow editor

---

## 🤝 Contributing

---

## 🤝 Contributing

We welcome contributions! This project thrives on community input.

### Ways to Contribute

- 🐛 **Report bugs**: [Open an issue](https://github.com/your-org/k8s-agent-stack/issues/new?template=bug_report.md)
- 💡 **Suggest features**: [Request a feature](https://github.com/your-org/k8s-agent-stack/issues/new?template=feature_request.md)
- 📖 **Improve docs**: Fix typos, add examples, clarify instructions
- 🤖 **Add agent examples**: Share your agent templates
- 🔧 **Enhance tooling**: Improve installer, add debugging helpers
- 🧪 **Add tests**: Integration tests, performance benchmarks

### Contribution Areas

| Area | What We Need | Difficulty |
|------|-------------|------------|
| **Agent Frameworks** | LangGraph, CrewAI, AutoGen integration | Medium |
| **Documentation** | Tutorials, troubleshooting guides, video walkthroughs | Easy |
| **Example Agents** | Domain-specific agent templates | Easy-Medium |
| **Monitoring** | Grafana dashboards, alerting rules | Medium |
| **CI/CD** | GitHub Actions workflows, testing automation | Medium |
| **Security** | RBAC policies, secret management patterns | Hard |

### Development Workflow

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/k8s-agent-stack.git
cd k8s-agent-stack

# 2. Create a branch
git checkout -b feature/my-awesome-feature

# 3. Make changes and test
./knative_orbstack.sh --debug
# Test your changes

# 4. Commit with clear messages
git commit -m "feat: add LangGraph deployment support"

# 5. Push and create PR
git push origin feature/my-awesome-feature
# Open PR on GitHub
```

### Commit Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding tests
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

---

## 📞 Community & Support

### Get Help

- 📚 **Documentation**: Start with [docs/](docs/) directory
- 🐛 **Issues**: [GitHub Issues](https://github.com/your-org/k8s-agent-stack/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/your-org/k8s-agent-stack/discussions)
- 📖 **Guides**: See [complete guide list](#-documentation-navigator)

### External Resources

- **Knative**: [knative.dev/docs](https://knative.dev/docs/)
- **kagent**: [github.com/kagent-dev/kagent](https://github.com/kagent-dev/kagent)
- **Google ADK**: [cloud.google.com/products/agent-developer-toolkit](https://cloud.google.com/products/agent-developer-toolkit)
- **Contour**: [projectcontour.io/docs](https://projectcontour.io/docs/)
- **Kubernetes**: [kubernetes.io/docs](https://kubernetes.io/docs/)

### Related Projects

- **[Knative Serving](https://github.com/knative/serving)** - Serverless runtime
- **[kagent](https://github.com/kagent-dev/kagent)** - Agent orchestration
- **[LangChain](https://github.com/langchain-ai/langchain)** - Agent framework
- **[LangGraph](https://github.com/langchain-ai/langgraph)** - Graph-based agents
- **[CrewAI](https://github.com/joaomdmoura/crewAI)** - Multi-agent collaboration

---

## ⚖️ License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

**Copyright © 2025 [Raphaël MANSUY](https://www.linkedin.com/in/raphaelmansuy/)**

This project is licensed under the Apache License, Version 2.0. You may obtain a copy of the License at:

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

---

## 🙏 Acknowledgments

This project builds on excellent open-source foundations:

- **[Knative](https://knative.dev/)** - Kubernetes-based serverless platform
- **[kagent](https://github.com/kagent-dev/kagent)** - Agent orchestration framework
- **[Google ADK](https://cloud.google.com/products/agent-developer-toolkit)** - Agent development toolkit
- **[Contour](https://projectcontour.io/)** - Envoy-based ingress controller

Special thanks to the communities behind these projects for their incredible work.

---

<div align="center">

## 🚀 k8s-agent-stack

**From zero to production AI agents in 5 minutes**

[Get Started](#-quick-install-5-minutes) • [Documentation](docs/) • [Examples](examples/) • [Contributing](#-contributing)

Created by [Raphaël MANSUY](https://www.linkedin.com/in/raphaelmansuy/) • Licensed under [Apache 2.0](LICENSE)

</div>
- [ ] AutoGen compatibility
- [ ] Multi-agent orchestration patterns
- [ ] Agent marketplace/catalog
- [ ] Enhanced observability dashboards

### Future
- [ ] LLM gateway with cost optimization
- [ ] Agent versioning and rollback strategies
- [ ] Multi-cloud deployment support
- [ ] Agent performance optimization toolkit

## Contributing
## Contributing

We welcome contributions! Areas where you can help:
- **New agent frameworks**: Add support for LangGraph, CrewAI, AutoGen, etc.
- **Documentation**: Improve guides, add tutorials, document patterns
- **Examples**: Share agent templates and use cases
- **Tooling**: Enhance installer, add debugging helpers, CI/CD templates
- **Testing**: Add integration tests, performance benchmarks

Please open an issue first to discuss major changes.

## Community & Support

- **Issues**: [GitHub Issues](https://github.com/your-org/k8s-agent-stack/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/k8s-agent-stack/discussions)
- **Documentation**: [docs/](docs/)

## License

MIT License - see LICENSE file for details

## Acknowledgments

Built on top of:
- [Knative Serving](https://knative.dev/docs/serving/) - Kubernetes-based serverless platform
- [kagent](https://github.com/kagent-dev/kagent) - Agent orchestration framework
- [Google ADK](https://cloud.google.com/products/agent-developer-toolkit) - Agent Developer Kit
- [Contour](https://projectcontour.io/) - Envoy-based ingress controller

---

**k8s-agent-stack** — From zero to production AI agents in minutes.

Enjoy — this repo is intentionally opinionated so you can iterate fast and focus on building great agents, not infrastructure.
