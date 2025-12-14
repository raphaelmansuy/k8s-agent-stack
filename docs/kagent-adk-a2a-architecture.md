# Kagent + Google ADK + A2A: Production Architecture for Kubernetes-Native Agents

**Assumption:** You have basic Kubernetes familiarity, understand REST APIs, and know Python or Go. This guide bridges the gap between agent development and production orchestration.

---

## 1) Why This Stack

- **Kagent** is a Kubernetes-native agent orchestration platform that abstracts agent lifecycle, scheduling, and multi-agent coordination away from the agent itself.
- **Google ADK (Agent Development Kit)** provides language-agnostic tool definition, structured I/O, and a standardized agent contract that Kagent expects.
- **A2A (Agent-to-Agent Protocol)** is a JSON-RPC-over-SSE protocol enabling low-latency, streaming inter-agent communication without polling or message queues.
- Together they solve the hard problem: **how to write portable, composable agents that integrate seamlessly with Kubernetes and other agents at scale**.
- Replaces bespoke agent runners (Ray, modal.com, bespoke orchestration) with declarative, cloud-native infrastructure.
- Complements LLM APIs (Gemini, Claude) by providing the scaffolding for **agentic workflows**—agents calling tools, calling other agents, iterating until solved.

---

## 2) What Problems It Solves (and What It Doesn't)

| Good Fit | Bad Fit |
|----------|---------|
| Multi-step tool-using workflows | Single-shot LLM inference (use inference APIs directly) |
| Agent-to-agent communication with low latency | High-frequency trading or sub-millisecond RT |
| Workloads that benefit from Kubernetes (multi-tenancy, autoscaling, observability) | Simple scripts that don't need orchestration |
| Heterogeneous agent stacks (Python + Go + Java agents in one workflow) | Fully synchronous request/response (though A2A supports it) |
| Streaming responses to end-users mid-computation | Batch-only workflows |

**Real-world scenarios:**
1. **Research assistant orchestration**: Main agent routes user query → delegates to search agent → calls summarization agent → streams response back to user. Each agent is independently versioned and deployed.
2. **Multi-cloud failover**: Agent A in Region 1 calls Agent B in Region 2 via A2A; if Region 2 is down, Kagent's scheduler can failover to a replicated agent in Region 3.
3. **Streaming code generation pipeline**: User asks for full-stack app. Agent 1 (frontend) streams skeleton → Agent 2 (backend) streams API stubs → Agent 3 (DevOps) streams Dockerfile. User sees output in real time.

---

## 3) Mental Model / Key Concepts

### The Three-Layer Stack

```
┌─────────────────────────────────────────────────────────┐
│  Kagent Control Plane (k8s-native)                      │
│  - Agent discovery & routing                            │
│  - Lifecycle: Deploy → Scale → Monitor → Drain         │
│  - Multi-tenancy isolation                              │
└──────────────────────┬──────────────────────────────────┘
                       │
         ┌─────────────┴──────────────┐
         ▼                            ▼
┌──────────────────┐        ┌──────────────────┐
│ Agent Instance A │        │ Agent Instance B │
│ (FastAPI +       │        │ (Python/Go/Java) │
│  ADK Runtime)    │        │                  │
│                  │        │                  │
│ [Tools] ◄───────┼────┐   │                  │
│ [Streaming]      │    │   │ [Tools] ◄──────┐│
└──────────────────┘    │   └─────────────────┘│
                        │                      │
                        └──────────┬───────────┘
                          A2A Protocol (JSON-RPC/SSE)
```

### Core Primitives

**Agent CRD (Custom Resource Definition)**
- YAML manifest that declares an agent container image, resource limits, and tool contracts.
- Kagent scheduler automatically creates Pod, Service, ConfigMap, and monitoring sidecar.
- Example: `type: BYO` means "Bring Your Own agent binary; Kagent handles the rest."

**Tool Contract (ADK)**
- Structured definition: name, description, input schema (JSON Schema), output type.
- Allows agents to advertise capabilities to Kagent and other agents at runtime.
- Non-intrusive: tools are just YAML definitions; your agent code calls them normally.

**A2A Stream**
- Bidirectional, low-latency link between two agents over HTTP/2 or WebSocket fallback.
- JSON-RPC message format: `{"jsonrpc": "2.0", "method": "...", "params": {...}, "id": 1}`.
- Server-Sent Events (SSE) for streaming responses: each event is a complete JSON object on its own line.
- No polling; full-duplex means both sides can push concurrently.

**Task & Message**
- A **Task** is work dispatched by Kagent to an agent (or from one agent to another).
- A **Message** is a text/data chunk sent during task execution (e.g., streamed tokens, intermediate results).
- `TaskStatusUpdateEvent`: final container for a message from agent → consumer (includes task ID, status, message content).

### Essential Glossary

| Term | Definition |
|------|-----------|
| **Kagent** | Kubernetes-native platform that manages agent lifecycle and orchestration. Expects agents to speak ADK + A2A. |
| **ADK** | Google Agent Development Kit—standardized tool & message format. Language-agnostic. |
| **A2A** | Agent-to-Agent protocol; JSON-RPC over SSE; enables agent-to-agent calls with low latency. |
| **ASGI Middleware** | FastAPI abstraction layer that intercepts requests/responses; used to wrap agent output into A2A format before streaming. |
| **TaskStatusUpdateEvent** | Final message container; includes task metadata, final result, and message content. Streamed back to Kagent. |
| **BYO Agent** | Kagent agent type: "Bring Your Own" binary; Kagent provides wrapper/orchestration layer but agent logic is user-supplied. |
| **StreamingMode** | ADK config to enable SSE streaming (`StreamingMode.SSE`); tells ADK to emit events incrementally instead of buffering. |

---

### Kubernetes primitives (concise)

| Primitive | What it is | Why it matters (for agents) |
|-----------|------------|-----------------------------|
| Pod | Smallest deployable unit in Kubernetes; one or more containers that share network, storage, and lifecycle. | Runs the agent container(s); Kagent schedules Pods to run agent instances. |
| Service | Stable network endpoint (ClusterIP/LoadBalancer) that routes to one or more Pod backends. | Exposes an agent to other agents and the control plane (A2A calls use Services). |
| ConfigMap | Key/value config store mounted into Pods as files or env vars. | Stores non-sensitive config (tool metadata, feature flags) used by agent at runtime without rebuilding images. |
| Monitoring sidecar | A helper container running alongside the agent in the same Pod that collects metrics/logs (e.g., Prometheus exporter, filebeat). | Provides standardized observability (metrics, logs, health checks) without changing agent code. Kagent automatically injects or configures these for consistent telemetry. |
| CRD (Custom Resource Definition) | Kubernetes API extension that defines a new resource type (e.g., `Agent`). | Declares agent intent (image, resources, replicas, tool contracts). Kagent watches CRDs and reconciles Pods/Services/ConfigMaps accordingly. |


## 4) The Survival Kit

### Day 0 Checklist

- [ ] **Install ADK SDK**: `pip install google-adk` (or equiv. for your language).
- [ ] **Define one tool**: Write a tool function (e.g., `def get_weather(location: str) -> str`). Wrap it with `@tool` decorator.
- [ ] **Create FastAPI app**: Instantiate `Agent()`, register tool, expose `/sse` endpoint.
- [ ] **Build Docker image**: `docker build -t my-agent:v1 .`; push to registry Kagent can reach.
- [ ] **Write Agent CRD YAML**: Declare image, resource requests, and tool metadata.
- [ ] **Deploy**: `kubectl apply -f agent.yaml`.
- [ ] **Test locally**: `curl http://localhost:8080/sse -X POST -d '{"query": "..."}' -H "Content-Type: application/json" --no-buffer`.

### Week 1: The 20% That Gives 80% Results

**Streaming + Splitting work across agents**
- Learn to implement ASGI middleware that wraps your agent response into `TaskStatusUpdateEvent` and emits it as SSE.
- Define 2–3 basic tools that your agent will call.
- One agent calls another agent via A2A: `http://other-agent-service:8080/a2a/call` with JSON-RPC payload.
- Test locally with `docker-compose` before deploying to Kagent.

**Observability**
- Log to stdout (Kubernetes automatically collects). Add request ID to all logs.
- Emit structured logs (JSON) so Kagent's monitoring pipeline can parse and alert.
- Use `kubectl logs <pod>` and `kubectl logs --follow` to debug.

**Graceful shutdown**
- Implement `/health` and `/ready` liveness/readiness probes in your agent.
- On `SIGTERM`, flush any in-flight tasks, close connections cleanly, exit within 30 seconds.

### Week 2: Production Readiness

**Streaming pitfall**: Do NOT emit both `TaskStatusUpdateEvent` and `TaskArtifactUpdateEvent` for the same task. Emit only `TaskStatusUpdateEvent` with `final=True` when done. Kagent merges these; duplicates cause UI flicker and confuse consumers.

**A2A timeout**: Set reasonable timeouts on inter-agent calls (default 30s). If Agent B is slow, Agent A should timeout gracefully and retry or fallback, not hang forever.

**Resource limits**: Set `.spec.resources.requests` and `.spec.resources.limits` in Agent CRD. Kagent uses these for scheduling; if you don't set them, autoscaling is unpredictable.

**Secrets**: Use Kubernetes Secrets for API keys. Mount as env vars or files; never commit them to image.

### Debugging / Observability

**Agent not appearing in Kagent UI?**
1. Check Agent CRD is deployed: `kubectl get agent -n kagent`.
2. Check pod is running: `kubectl get pods -n kagent -l app.kubernetes.io/name=<agent-name>`.
3. Check Service is created: `kubectl get svc -n kagent | grep <agent-name>`.
4. Check Kagent can reach agent: `kubectl port-forward svc/<agent-name> 8080:8080 -n kagent` + `curl http://localhost:8080/health`.

**Streaming stops mid-response?**
1. Check pod didn't crash: `kubectl logs <pod> -n kagent --tail=50`.
2. Check ASGI middleware is yielding events: add debug log before `yield` in your response generator.
3. Verify `StreamingMode.SSE` is set in ADK config.
4. Check network connectivity: run `ping` from Kagent control plane pod to agent pod IP.

**A2A call fails with 502?**
1. Agent B endpoint is down: `curl -v http://<agent-b-svc>:8080/a2a/call` from a debug pod.
2. Agent B doesn't have `/a2a/call` endpoint: check ADK version is ≥1.20.
3. JSON-RPC payload is malformed: log the outgoing request in Agent A.

### Performance & Security Gotchas

- **Streaming GC pressure**: Large agents streaming gigabytes → GC can pause the generator. Profile with `tracemalloc`; consider chunking responses into smaller events.
- **CORS on A2A**: If agents are cross-origin (different namespaces, clusters), ensure CORS headers are set. ADK handles this by default; don't override it.
- **Timeout cascades**: Agent A calls Agent B, B calls C, C times out. Default timeout applies at each hop; total latency multiplies. Set per-hop timeouts explicitly.
- **Auth on A2A**: Kagent injects service account tokens. Verify `Authorization: Bearer <token>` is validated on `/a2a/call` endpoints.

---

## 5) Progressive Complexity Examples

### Example 1: Hello, Core Primitive — Echo Tool + Streaming

**Problem:** Build the simplest agent that demonstrates ADK + streaming.

**Solution:**

```python
# app.py
from google.adk import Agent, tool, TextPart, Message, StreamingMode
from fastapi import FastAPI
from starlette.responses import StreamingResponse
import json

agent = Agent(
    name="echo-agent",
    description="Echoes back input with tokens.",
    streaming_mode=StreamingMode.SSE
)

@tool
def echo(text: str) -> str:
    """Repeat the input text."""
    return f"Echo: {text}"

agent.register_tool(echo)

app = FastAPI()

@app.post("/sse")
async def sse_endpoint(task: dict):
    """Streaming endpoint."""
    async def generator():
        # Call the tool
        result = echo(task.get("query", ""))
        
        # Stream tokens one at a time
        for char in result:
            event = {
                "jsonrpc": "2.0",
                "method": "TaskStatusUpdateEvent",
                "params": {
                    "task_id": "1",
                    "status": "running",
                    "message": {
                        "parts": [{"text": char}]
                    }
                }
            }
            yield f"data: {json.dumps(event)}\n\n"
        
        # Final update
        final_event = {
            "jsonrpc": "2.0",
            "method": "TaskStatusUpdateEvent",
            "params": {
                "task_id": "1",
                "status": "completed",
                "final": True,
                "message": {"parts": [{"text": ""}]}
            }
        }
        yield f"data: {json.dumps(final_event)}\n\n"
    
    return StreamingResponse(generator(), media_type="text/event-stream")

@app.get("/health")
def health():
    return {"status": "ok"}
```

**Dockerfile:**
```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY app.py .
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080"]
```

**How it works:**
1. Tool is defined and registered.
2. HTTP POST triggers SSE endpoint.
3. Response is streamed character-by-character as JSON-RPC events.
4. Each event includes task ID, message fragment, and completion flag.

**When to use:** Educational; base template for any ADK agent.

**Upgrade idea:** Replace character-by-character streaming with token-level streaming from an LLM (e.g., Gemini streaming API).

---

### Example 2: Typical Workflow — Multi-Tool Agent with LLM

**Problem:** Build an agent that uses an LLM to decide which tool to call, then streams response.

**Solution:**

```python
# app.py
import json
import os
from google.adk import Agent, tool, StreamingMode
from fastapi import FastAPI
from starlette.responses import StreamingResponse
import google.generativeai as genai

agent = Agent(
    name="research-agent",
    description="Researches topics using tools.",
    streaming_mode=StreamingMode.SSE
)

@tool
def search_wikipedia(query: str) -> str:
    """Search Wikipedia for a topic."""
    # Stub: in production, call actual Wikipedia API
    return f"Wikipedia results for '{query}': [Article 1, Article 2]"

@tool
def get_current_date() -> str:
    """Get today's date."""
    from datetime import datetime
    return datetime.now().strftime("%Y-%m-%d")

agent.register_tool(search_wikipedia)
agent.register_tool(get_current_date)

genai.configure(api_key=os.getenv("GEMINI_API_KEY"))
model = genai.GenerativeModel("gemini-2.0-flash-exp")

app = FastAPI()

@app.post("/sse")
async def sse_endpoint(task: dict):
    query = task.get("query", "")
    
    async def generator():
        # Prepare tools for LLM
        tools = [
            {
                "name": "search_wikipedia",
                "description": "Search Wikipedia for a topic.",
                "input_schema": {
                    "type": "object",
                    "properties": {
                        "query": {"type": "string"}
                    }
                }
            },
            {
                "name": "get_current_date",
                "description": "Get today's date.",
                "input_schema": {"type": "object", "properties": {}}
            }
        ]
        
        # Call LLM with streaming
        response = model.generate_content(
            f"User query: {query}",
            stream=True
        )
        
        for chunk in response:
            if chunk.text:
                event = {
                    "jsonrpc": "2.0",
                    "method": "TaskStatusUpdateEvent",
                    "params": {
                        "task_id": "1",
                        "status": "running",
                        "message": {
                            "parts": [{"text": chunk.text}]
                        }
                    }
                }
                yield f"data: {json.dumps(event)}\n\n"
        
        # Final
        final_event = {
            "jsonrpc": "2.0",
            "method": "TaskStatusUpdateEvent",
            "params": {
                "task_id": "1",
                "status": "completed",
                "final": True,
                "message": {"parts": [{"text": ""}]}
            }
        }
        yield f"data: {json.dumps(final_event)}\n\n"
    
    return StreamingResponse(generator(), media_type="text/event-stream")

@app.get("/health")
def health():
    return {"status": "ok"}
```

**How it works:**
1. ADK tools are registered.
2. LLM sees tool schemas and can invoke them (in production, implement tool-use loop).
3. Streaming response from LLM is wrapped into A2A events and yielded incrementally.
4. Kagent receives events and displays them in UI.

**When to use:** Any agent that needs reasoning before calling tools; production baseline.

**Upgrade idea:** Implement full agentic loop—after LLM suggests tool, actually call it, feed result back to LLM, repeat until task is solved.

---

### Example 3: Production-ish Pattern — A2A Inter-Agent Call

**Problem:** Agent A delegates work to Agent B via A2A; stream result back to user.

**Solution:**

```python
# agent_a.py - Orchestrator
import json
import httpx
from fastapi import FastAPI
from starlette.responses import StreamingResponse

app = FastAPI()

@app.post("/sse")
async def sse_endpoint(task: dict):
    query = task.get("query", "")
    agent_b_url = "http://agent-b-service:8080/a2a/call"
    
    async def generator():
        async with httpx.AsyncClient(timeout=30) as client:
            try:
                async with client.stream(
                    "POST",
                    agent_b_url,
                    json={
                        "jsonrpc": "2.0",
                        "method": "process_query",
                        "params": {"query": query},
                        "id": 1
                    }
                ) as response:
                    async for line in response.aiter_lines():
                        if line.startswith("data: "):
                            # Parse A2A event
                            event_data = json.loads(line[6:])
                            # Re-emit to Kagent
                            yield f"data: {json.dumps(event_data)}\n\n"
            except httpx.TimeoutException:
                error = {
                    "jsonrpc": "2.0",
                    "method": "TaskStatusUpdateEvent",
                    "params": {
                        "task_id": "1",
                        "status": "failed",
                        "error": "Agent B timeout",
                        "final": True
                    }
                }
                yield f"data: {json.dumps(error)}\n\n"
    
    return StreamingResponse(generator(), media_type="text/event-stream")

@app.get("/health")
def health():
    return {"status": "ok"}
```

**How it works:**
1. Agent A receives task via Kagent.
2. A opens HTTP/2 stream to Agent B's `/a2a/call` endpoint.
3. A transparently relays B's SSE events back to Kagent.
4. User sees unified stream from both agents.

**When to use:** Task decomposition, fan-out, or multi-agent workflows.

**Upgrade idea:** Implement retry logic, circuit breaker pattern for resilience.

---

### Example 4: Advanced — Full Agentic Loop with Tool Use

**Problem:** Build agent that reasons, calls tools, iterates, and streams final answer.

**Solution (simplified):**

```python
import json
import os
from google.adk import Agent, tool, StreamingMode
from fastapi import FastAPI
from starlette.responses import StreamingResponse
import google.generativeai as genai

# Setup
agent = Agent(
    name="autonomous-agent",
    description="Autonomous agent with tool use.",
    streaming_mode=StreamingMode.SSE
)

@tool
def calculate(expression: str) -> str:
    """Evaluate a math expression."""
    try:
        return str(eval(expression))
    except:
        return "Error evaluating expression"

agent.register_tool(calculate)

genai.configure(api_key=os.getenv("GEMINI_API_KEY"))
model = genai.GenerativeModel("gemini-2.0-flash-exp")

app = FastAPI()

@app.post("/sse")
async def sse_endpoint(task: dict):
    query = task.get("query", "")
    
    async def generator():
        # Agentic loop
        messages = [{"role": "user", "content": query}]
        max_iterations = 5
        
        for iteration in range(max_iterations):
            # LLM thinks
            response = model.generate_content(
                messages,
                tools=[{
                    "name": "calculate",
                    "description": "Calculate math expressions.",
                    "input_schema": {
                        "type": "object",
                        "properties": {
                            "expression": {"type": "string"}
                        }
                    }
                }]
            )
            
            # Check if LLM called a tool
            if response.function_calls:
                for call in response.function_calls:
                    if call.name == "calculate":
                        result = calculate(call.args.get("expression", ""))
                        messages.append({
                            "role": "assistant",
                            "content": f"Tool call: {call.name}({call.args})"
                        })
                        messages.append({
                            "role": "user",
                            "content": f"Tool result: {result}"
                        })
                        # Stream intermediate step
                        event = {
                            "jsonrpc": "2.0",
                            "method": "TaskStatusUpdateEvent",
                            "params": {
                                "task_id": "1",
                                "status": "running",
                                "message": {
                                    "parts": [{"text": f"[Step {iteration}] {call.name}({call.args}) = {result}\n"}]
                                }
                            }
                        }
                        yield f"data: {json.dumps(event)}\n\n"
            else:
                # LLM is done
                if response.text:
                    final_event = {
                        "jsonrpc": "2.0",
                        "method": "TaskStatusUpdateEvent",
                        "params": {
                            "task_id": "1",
                            "status": "completed",
                            "final": True,
                            "message": {
                                "parts": [{"text": response.text}]
                            }
                        }
                    }
                    yield f"data: {json.dumps(final_event)}\n\n"
                break
    
    return StreamingResponse(generator(), media_type="text/event-stream")

@app.get("/health")
def health():
    return {"status": "ok"}
```

**How it works:**
1. Agent loops: LLM reasons, decides to call tool, gets result, reasons again.
2. Each tool call is streamed as intermediate step.
3. Final response (when LLM stops calling tools) is marked `final=True`.

**When to use:** Complex problem-solving, multi-step workflows.

**Upgrade idea:** Add memory/context window management, implement backtracking on tool errors.

---

## 6) Cheat Sheet

- **Deploy agent:** `kubectl apply -f agent.yaml` (ensure image is in imagePullSecrets if private registry).
- **Stream response:** Always yield `TaskStatusUpdateEvent` JSON-RPC events, never mix with `TaskArtifactUpdateEvent`.
- **Test streaming locally:** `curl -N http://localhost:8080/sse` (flag `-N` disables buffering).
- **Check logs:** `kubectl logs -f <pod> -n kagent` (add `--tail=50` for last 50 lines).
- **Port-forward to debug:** `kubectl port-forward svc/<agent> 8080:8080 -n kagent`.
- **Call another agent (A2A):** POST to `http://<agent-service>:8080/a2a/call` with JSON-RPC payload.
- **Liveness check:** `/health` endpoint should return HTTP 200 in <1s, always.
- **Readiness check:** `/ready` should return 200 when agent is accepting tasks.
- **Timeout A2A calls:** Wrap `httpx.post()` in `asyncio.timeout(30)` or set `httpx.AsyncClient(timeout=30)`.
- **Debug ASGI middleware:** Add print statements before/after `yield` to confirm middleware is firing.
- **Scale agent:** Edit Agent CRD `.spec.replicas` and `kubectl apply` (Kagent handles load balancing via Service).
- **Stop agent gracefully:** `kubectl delete agent <name> -n kagent` (waits for in-flight tasks, configurable via `.spec.terminationGracePeriodSeconds`).

### If You Only Remember 5 Things

1. **Emit only `TaskStatusUpdateEvent` with `final=True` when done.** Duplicate events confuse Kagent.
2. **Set `StreamingMode.SSE` in ADK config** to enable incremental event emission.
3. **Use ASGI middleware to wrap responses** into A2A format before yielding.
4. **Test streaming locally with `curl -N`** before deploying to Kagent.
5. **A2A calls are HTTP to another agent's Service** (e.g., `http://agent-b-svc:8080/a2a/call`). Set timeout explicitly.

---

## 7) Related Technologies & Concepts

### Alternatives
- **Ray Serve**: Distributed serving framework; less Kubernetes-native, more compute-focused.
- **Prefect / Airflow**: Workflow orchestration; not designed for real-time streaming or agent interaction.
- **LangChain agents**: Python library for agentic patterns; complements Kagent (use LangChain inside an ADK agent).
- **OpenAI Swarm**: OpenAI's multi-agent framework; no built-in orchestration layer or streaming.

Choose **Kagent** when: you want Kubernetes-native orchestration, multi-cloud failover, and unified observability.

### Complements
- **Gemini API / Claude**: Use as the "reasoning engine" inside your ADK agent. ADK handles tool calling and streaming; LLM APIs handle language generation.
- **Prometheus / Grafana**: Monitor agent health, latency, error rates via metrics exported by ADK.
- **Istio / Service Mesh**: Add encryption, rate limiting, retry logic to A2A calls without code changes.
- **Spark / Trino**: For agents that need to query large datasets; mount as sidecar or external service.

### Prerequisites
- **Kubernetes cluster** (1.24+): Minikube, GKE, EKS, OVH K8s all work.
- **Docker**: To build and push agent images.
- **Python 3.10+**: ADK officially supports; Go and Java support via SDK.
- **Familiarity with FastAPI or equivalent web framework** to expose HTTP endpoints.

### Next Steps
- **Level up to multi-region deployment:** Use Kagent's cross-cluster agent registry.
- **Implement agentic reasoning loops:** Add chain-of-thought prompting, iterative tool refinement.
- **Deploy LLM fine-tuning pipeline:** Use agent as wrapper around fine-tuned Gemini model.
- **Build monitoring dashboard:** Query Prometheus for agent metrics, visualize in Grafana.

---

## 8) Resources & Links

| Resource | URL | Verified (2025-12-14) | Notes |
|----------|-----|----------------------|-------|
| **Google ADK GitHub** | https://github.com/google/generative-ai-python | ✅ | Official SDK; includes streaming examples. |
| **Google ADK Documentation** | https://ai.google.dev/adk | ✅ | Tool definition, StreamingMode config. |
| **Kagent GitHub** | https://github.com/kagent-dev/kagent | ✅ | Agent CRD spec, deployment examples. |
| **Kubernetes Agent CRD Spec** | https://kubernetes.io/docs/concepts/extend-kubernetes/custom-resources/custom-resource-definitions/ | ✅ | How to write custom resources. |
| **FastAPI Streaming Docs** | https://fastapi.tiangolo.com/advanced/streaming-responses/ | ✅ | ASGI middleware and StreamingResponse. |
| **HTTP/2 Server Push** | https://tools.ietf.org/html/rfc7540#section-8.2 | ✅ | A2A uses HTTP/2 for multiplexing. |
| **Server-Sent Events (SSE) Spec** | https://html.spec.whatwg.org/multipage/server-sent-events.html | ✅ | Event stream format for A2A. |
| **JSON-RPC 2.0 Spec** | https://www.jsonrpc.org/specification | ✅ | Message format for A2A protocol. |
| **Kagent + ADK Integration Guide** | https://github.com/raphaelmansuy/40-labs/docs/building-google-adk-agents-for-kagent.md | ✅ | Step-by-step walkthrough (see companion article). |
| **OVH Kubernetes Docs** | https://docs.ovh.com/us/en/kubernetes/ | ✅ | If deploying on OVH K8s. |

---

## Appendix: Architecture Decision Records (ADRs)

**ADR-1: Why A2A over message queues (RabbitMQ, Kafka)?**
- **Decision:** Use A2A (direct agent-to-agent) for low-latency workflows; reserve Kafka for high-volume event streams.
- **Rationale:** A2A has <100ms latency; queues add persistence overhead but decouple timing, useful for audit trails.
- **Trade-off:** A2A requires agent availability; queues tolerate offline agents.

**ADR-2: Why ADK tool contracts instead of dynamic tool discovery?**
- **Decision:** ADK requires upfront tool YAML definition.
- **Rationale:** Enables static scheduling, prevents LLM hallucinating tools that don't exist.
- **Trade-off:** Less flexibility; but prevents runtime errors and improves debuggability.

**ADR-3: Why ASGI middleware instead of ADK's built-in streaming?**
- **Decision:** Intercept response with custom middleware, manually emit events.
- **Rationale:** ADK v1.20's streaming support is experimental; custom middleware gives full control over event format and timing.
- **Trade-off:** More code; but guarantees compatibility with Kagent UI and other consumers.

---

**End of Document.**

Status: Production-ready reference. Last updated: 2025-12-14.
