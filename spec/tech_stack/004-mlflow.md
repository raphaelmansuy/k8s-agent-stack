# Why MLflow for GenAI and AI Agents

> ⚠️ **CRITICAL: Agent Safety & Evaluation**
> 
> MLflow evaluation is the **primary quality gate** for agent deployment in AgentStack.
> No agent can be deployed without passing MLflow safety and correctness evaluations.
> See [010-agent-evaluation.md](../010-agent-evaluation.md) for complete evaluation spec.

- Unifies ML and GenAI lifecycles: Tracks experiments, evaluates LLMs/agents, and deploys via AI Gateway, reducing tool sprawl in agentic workflows.
- Enables observability for opaque GenAI: Tracing captures prompts, tool calls, and retrievals in agents, speeding debugging of multi-step reasoning.
- Built-in LLM judges for evaluation: Automates quality checks like correctness or hallucination, replacing custom scripts for faster iteration on agents.
- Framework-agnostic: Integrates with LangChain, LlamaIndex, etc., complementing agent builders without lock-in.
- AI Gateway standardizes LLM access: Manages keys, rate limits, and unified endpoints for agents calling multiple providers like OpenAI or Anthropic.
- Scales to production: Versions prompts, agents, and apps with metrics, shining in enterprise GenAI where reproducibility matters.
- Open-source and extensible: Avoids vendor silos, complements MLOps stacks like Databricks or SageMaker for GenAI agents.

# What problems it solves (and what it doesn't)

| Good fit | Bad fit |
|----------|---------|
| Evaluating LLM/agent quality systematically (e.g., hallucination, relevance) without building eval pipelines from scratch. | Real-time inference serving at massive scale—use dedicated servers like vLLM instead. |
| Tracing complex agent workflows to debug tool calls, retrievals, and multi-LLM interactions. | Training LLMs from scratch—MLflow focuses on lifecycle post-training. |
| Managing prompts, configs, and versions in agentic apps for reproducibility. | Non-GenAI ML tasks where classic MLflow suffices without GenAI extensions. |

Real-world scenarios:
- Chatbot agent failing on edge cases: Use tracing to pinpoint bad tool calls and evaluation to score responses, iterating quickly.
- Multi-provider agent (e.g., OpenAI + Claude): AI Gateway unifies APIs, simplifying routing and key management.
- Enterprise RAG agent: Log traces, evaluate retrieval accuracy, and version prompts to ensure compliance and performance.

# Mental model / key concepts (the minimum to think correctly)
MLflow for GenAI treats agents and apps as traceable, evaluable entities in a unified lifecycle: experiment → evaluate → monitor → deploy. Core primitives:
- **Tracing**: Logs spans (inputs/outputs/metadata) for each step in an agent (e.g., prompt → tool call → response).
- **Evaluation**: Uses datasets, predict functions, and scorers (LLM judges) to measure quality metrics like correctness.
- **AI Gateway**: Proxy for LLM calls, handling routing, auth, and unification.
- **Prompt Registry**: Versions templates as models for reuse in agents.
- **ChatModel/PyFunc**: Wrappers to log/deploy custom agents as MLflow models.

Interaction: Traces feed into evaluations for quality gates; evaluations inform versioning; Gateway deploys versioned agents.

ASCII diagram (GenAI Agent Lifecycle):

```
[Experiment] --> Trace spans (prompts, tools) --> Log to MLflow
                 |
                 v
[Evaluate] --> Dataset + Predict Fn + Scorer --> Metrics (e.g., hallucination score)
                 |
                 v
[Monitor/Deploy] --> Version agent --> AI Gateway --> Production calls
```

Glossary:
- Span: A logged unit of work (e.g., one tool call).
- Scorer: LLM-based judge for metrics (e.g., Correctness).
- Predict Fn: Function generating outputs from inputs for eval.
- Autolog: One-line tracing enablement for libraries like LangChain.

# The survival kit (actionable fastest path to proficiency)
Prioritized checklist:
- **Day 0**: Install MLflow (`pip install mlflow>=3.0`), set up tracking server (`mlflow server`), log a simple LLM call with tracing.
- **Week 1**: Build/eval a basic agent: Create dataset, define scorers, run `mlflow.genai.evaluate()`, trace with `@mlflow.trace`.
- **Week 2**: Deploy via AI Gateway: Set up routes, version prompts/agents, monitor in UI.

20% of features for 80% results: Tracing (autolog for LangChain), evaluation (`mlflow.genai.evaluate()` with built-in scorers), AI Gateway for deployment.

Common pitfalls + avoids:
- Overlooking tracing overhead: Use sampling to reduce logs; avoid in prod inference.
- Bad eval datasets: Start small, include ground truth; iterate based on UI results.
- Hardcoded configs: Use `model_config` in logging for flexibility.

Debugging/observability tips: View traces in MLflow UI for span hierarchies; filter by attributes like model name; correlate with eval metrics.

Performance gotchas: Tracing adds latency—disable in high-throughput paths; AI Gateway rate limits prevent over-billing.
Security gotchas: Sanitize traces to avoid logging PII; use Gateway's key management for secure LLM access.

# Progressive complexity examples (high value, minimal but real)
## Example 1: "Hello, core primitive" (log a simple LLM call)
Problem: Track a basic OpenAI prompt-response for reproducibility.

Solution (code):
```python
import mlflow
import openai

mlflow.start_run()
client = openai.OpenAI()
response = client.chat.completions.create(model="gpt-4o-mini", messages=[{"role": "user", "content": "Hello, MLflow!"}])
mlflow.log_param("model", "gpt-4o-mini")
mlflow.log_text(response.choices[0].message.content, "response.txt")
mlflow.end_run()
```

How it works: Starts a run, calls LLM, logs params/artifacts. View in UI for quick inspection.

When to use: Initial experiments with single prompts. Not for complex agents—add tracing.

Upgrade: Add autolog for automatic span capture: `mlflow.openai.autolog()`.

## Example 2: "Typical workflow" (evaluate an LLM)
Problem: Assess if an LLM answers questions correctly without hallucinations.

Solution (code):
```python
import mlflow
from mlflow.genai.scorers import Correctness

dataset = [{"inputs": {"question": "What is MLflow?"}, "expectations": {"expected_response": "ML lifecycle platform"}}]
def predict_fn(question): return openai.OpenAI().chat.completions.create(model="gpt-4o-mini", messages=[{"role": "user", "content": question}]).choices[0].message.content

results = mlflow.genai.evaluate(data=dataset, predict_fn=predict_fn, scorers=[Correctness()])
```

How it works: Defines dataset with expectations, predict function for outputs, scorer for judging. Runs eval, logs metrics to MLflow.

When to use: Validating LLM quality pre-deployment. Not for agents—extend with multi-step datasets.

Upgrade: Add custom scorer for domain-specific guidelines, e.g., `Guidelines(name="concise", guidelines="Response under 50 words")`.

ASCII diagram (Eval Flow):

```
Dataset --> Predict Fn --> Outputs
            |
            v
Scorer (LLM Judge) --> Metrics --> Log to Run
```

## Example 3: "Production-ish pattern" (trace a simple agent)
Problem: Debug a LangChain agent that retrieves and reasons.

Solution (code):
```python
import mlflow
from langchain.agents import initialize_agent
from langchain.llms import OpenAI

mlflow.langchain.autolog()
llm = OpenAI()
tools = [...]  # e.g., search tool
agent = initialize_agent(tools, llm)
agent.run("What is the weather?")
```

How it works: Autolog enables tracing; captures spans for LLM calls, tool invocations, outputs. View hierarchy in UI.

When to use: Observing agent chains. Not for custom agents—use manual `@mlflow.trace`.

Upgrade: Add sampling to trace only 10% of runs for scale: `mlflow.set_tracing_sampling_rate(0.1)`.

## Example 4: "Advanced but common" (deploy custom chat agent)
Problem: Version and deploy an agent that audits responses.

Solution (code):
```python
from mlflow.pyfunc import ChatModel
import mlflow

class AuditAgent(ChatModel):
    def predict(self, context, messages, params):
        # Audit logic with LLM call
        return {"choices": [{"message": {"content": "Audited response"}}]}

model_config = {"judge": {"endpoint": "gpt-4o"}}
with mlflow.start_run():
    mlflow.pyfunc.log_model(python_model=AuditAgent(), model_config=model_config)
```

How it works: Subclasses ChatModel, implements predict with logic. Logs with config for decoupling.

When to use: Standardizing agent interfaces for deployment. Not for non-chat agents—use PyFunc.

Upgrade: Integrate AI Gateway: Define route in config, deploy model URI to Gateway for unified serving.

# Cheat sheet
- Start tracking server: `mlflow server --port 5000`.
- Enable autolog: `mlflow.langchain.autolog()`.
- Log param: `mlflow.log_param("key", value)`.
- Trace function: `@mlflow.trace(name="span_name") def fn(): ...`.
- Start span: `with mlflow.start_span("name") as span: span.set_inputs(inputs)`.
- Eval dataset entry: `{"inputs": {"question": "q"}, "expectations": {"expected": "a"}}`.
- Built-in scorer: `Correctness()`.
- Custom scorer: `Guidelines(name="custom", guidelines="criteria")`.
- Run eval: `mlflow.genai.evaluate(data, predict_fn, scorers)`.
- Log model: `mlflow.pyfunc.log_model(python_model=Class(), model_config=dict)`.
- Load model: `loaded = mlflow.pyfunc.load_model(uri)`.
- Set sampling: `mlflow.set_tracing_sampling_rate(0.5)`.
- AI Gateway route: Define in YAML, e.g., `routes: openai: {provider: openai, api_key: env}`.
- View UI: Browser at `localhost:5000`.
- Prompt registry: Log as model with `mlflow.log_prompt(template)`.
- Sanitize traces: Use filters to remove sensitive data.

If you only remember 5 things:
- Autolog for easy tracing.
- `mlflow.genai.evaluate()` for quality checks.
- ChatModel for agent standardization.
- AI Gateway for LLM unification.
- UI for all observability.

# Related technologies & concepts (map of the neighborhood)
- **Alternatives**: Phoenix (tracing-focused, choose when MLflow's full lifecycle is overkill); LangSmith (LangChain-specific, choose for deep integration but risk lock-in).
- **Complements**: LangChain/LlamaIndex (build agents, then trace/eval with MLflow); Databricks/SageMaker (host MLflow for managed scaling).
- **Prereqs**: OpenTelemetry basics for tracing; LLM APIs like OpenAI.
- **Next steps**: MLflow Deployments for serving; advanced eval with RAGAS integration (choose when needing retrieval-specific metrics).

Assuming intermediate audience with Python/LLM basics; focusing on MLflow 3.x for GenAI agents.

| Resource | URL |
|----------|-----|
| Official MLflow Documentation | https://mlflow.org/docs/latest/index.html |
| MLflow GenAI Evaluation and Monitoring | https://mlflow.org/docs/latest/genai/eval-monitor/ |
| MLflow Tracing for LLM Observability | https://mlflow.org/docs/latest/genai/tracing/ |
| Custom GenAI Models Tutorial | https://mlflow.org/docs/latest/genai/flavors/chat-model-guide/ |
| MLflow for Generative AI Overview | https://mlflow.org/genai |
| Databricks MLflow for GenAI | https://docs.databricks.com/en/mlflow3/genai/overview/ |
| **AgentStack Evaluation Spec** | [010-agent-evaluation.md](../010-agent-evaluation.md) |