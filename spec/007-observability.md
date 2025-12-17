# 007 - Observability

> Metrics, Logging, Tracing, and Alerting

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Observability Stack

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Observability Stack                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Collection Layer                      │    │
│  │  OpenTelemetry Collector (metrics, logs, traces)        │    │
│  └─────────────────────────────────────────────────────────┘    │
│         │                    │                    │              │
│         ▼                    ▼                    ▼              │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐      │
│  │ Prometheus  │      │    Loki     │      │   Tempo     │      │
│  │  (Metrics)  │      │   (Logs)    │      │  (Traces)   │      │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘      │
│         │                    │                    │              │
│         └────────────────────┼────────────────────┘              │
│                              │                                   │
│                              ▼                                   │
│                    ┌─────────────────┐                          │
│                    │     Grafana     │                          │
│                    │  (Dashboards)   │                          │
│                    └─────────────────┘                          │
│                              │                                   │
│                              ▼                                   │
│                    ┌─────────────────┐                          │
│                    │   Alertmanager  │                          │
│                    │    (Alerts)     │                          │
│                    └─────────────────┘                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Component Selection

| Component | Purpose | License | Alternative |
|-----------|---------|---------|-------------|
| **OpenTelemetry** | Collection | Apache 2.0 | - |
| **Prometheus** | Metrics | Apache 2.0 | VictoriaMetrics |
| **Loki** | Logs | AGPL 3.0 | - |
| **Tempo** | Traces | AGPL 3.0 | Jaeger (Apache 2.0) |
| **Grafana** | Visualization | AGPL 3.0 | - |

**Note**: Loki/Tempo/Grafana are AGPL but free to use. For pure Apache 2.0: use Jaeger for traces.

---

## 2. Metrics

### 2.1 Key Metrics

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Agent Metrics                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  REQUEST METRICS                                                 │
│  • agent_requests_total{agent, status, method}                  │
│  • agent_request_duration_seconds{agent, quantile}              │
│  • agent_request_size_bytes{agent, direction}                   │
│                                                                  │
│  LLM METRICS                                                     │
│  • agent_tokens_total{agent, model, type}  # input/output       │
│  • agent_llm_latency_seconds{agent, model}                      │
│  • agent_llm_errors_total{agent, model, error_type}             │
│                                                                  │
│  TOOL METRICS                                                    │
│  • agent_tool_calls_total{agent, tool, status}                  │
│  • agent_tool_duration_seconds{agent, tool}                     │
│                                                                  │
│  SCALING METRICS                                                 │
│  • agent_replicas{agent}                                        │
│  • agent_cold_starts_total{agent}                               │
│  • agent_queue_depth{agent}                                     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Prometheus Recording Rules

```yaml
groups:
  - name: agent_slos
    interval: 30s
    rules:
      # Request rate
      - record: agent:request_rate:5m
        expr: sum(rate(agent_requests_total[5m])) by (agent)
      
      # Error rate
      - record: agent:error_rate:5m
        expr: |
          sum(rate(agent_requests_total{status=~"5.."}[5m])) by (agent)
          /
          sum(rate(agent_requests_total[5m])) by (agent)
      
      # P99 latency
      - record: agent:latency_p99:5m
        expr: |
          histogram_quantile(0.99, 
            sum(rate(agent_request_duration_seconds_bucket[5m])) by (agent, le)
          )
      
      # Token spend rate
      - record: agent:tokens_per_hour
        expr: sum(increase(agent_tokens_total[1h])) by (agent, model)
```

### 2.3 SLO Definitions

| SLI | Target | Measurement |
|-----|--------|-------------|
| **Availability** | 99.9% | `1 - error_rate` |
| **Latency P99** | < 5s | `latency_p99` |
| **Cold Start** | < 3s | `cold_start_duration_p95` |

---

## 3. Logging

### 3.1 Log Structure

```json
{
  "timestamp": "2025-01-15T10:30:00.123Z",
  "level": "info",
  "message": "Agent request completed",
  "service": "agent-runtime",
  "trace_id": "abc123xyz",
  "span_id": "def456",
  "agent_id": "agt_abc",
  "project_id": "prj_123",
  "request_id": "req_789",
  "duration_ms": 1250,
  "tokens": {
    "input": 150,
    "output": 342
  },
  "tools_called": ["search-kb"],
  "model": "gpt-4o",
  "status": "success"
}
```

### 3.2 Log Levels

| Level | Use Case | Retention |
|-------|----------|-----------|
| `error` | Failures, exceptions | 90 days |
| `warn` | Degraded performance, retries | 30 days |
| `info` | Request completion, deployments | 14 days |
| `debug` | Detailed execution flow | 3 days |

### 3.3 Log Aggregation

```yaml
# Loki configuration
loki:
  storage:
    type: s3
    bucket: agentstack-logs
  retention:
    enabled: true
    period: 90d
  ingestion:
    rate_limit: 10MB/s
    burst_limit: 50MB/s
  
# Log labels (for efficient querying)
labels:
  - project_id
  - agent_id
  - level
  - service
```

---

## 4. Distributed Tracing

### 4.1 Trace Context

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Request Trace                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  trace_id: abc123xyz                                            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Span: api-gateway                                        │    │
│  │ Duration: 1250ms                                         │    │
│  │ ├─────────────────────────────────────────────────────┐  │    │
│  │ │ Span: agent-runtime                                 │  │    │
│  │ │ Duration: 1200ms                                    │  │    │
│  │ │ ├─────────────────────────────────────────────────┐│  │    │
│  │ │ │ Span: llm-call (gpt-4o)                         ││  │    │
│  │ │ │ Duration: 800ms                                 ││  │    │
│  │ │ │ Attributes: tokens_in=150, tokens_out=342       ││  │    │
│  │ │ └─────────────────────────────────────────────────┘│  │    │
│  │ │ ├─────────────────────────────────────────────────┐│  │    │
│  │ │ │ Span: tool-call (search-kb)                     ││  │    │
│  │ │ │ Duration: 200ms                                 ││  │    │
│  │ │ └─────────────────────────────────────────────────┘│  │    │
│  │ └─────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 OpenTelemetry Instrumentation

```go
// Go instrumentation example
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (h *ChatHandler) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    ctx, span := otel.Tracer("agent-runtime").Start(ctx, "chat")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("agent_id", req.AgentID),
        attribute.String("model", req.Model),
    )
    
    // LLM call with child span
    ctx, llmSpan := otel.Tracer("agent-runtime").Start(ctx, "llm-call")
    response, err := h.llmClient.Complete(ctx, prompt)
    llmSpan.SetAttributes(
        attribute.Int("tokens_input", response.Usage.Input),
        attribute.Int("tokens_output", response.Usage.Output),
    )
    llmSpan.End()
    
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    return response, nil
}
```

### 4.3 Trace Sampling

```yaml
# Sampling configuration
sampling:
  default: 0.1          # 10% of requests
  rules:
    - name: errors
      condition: "status >= 400"
      sample_rate: 1.0   # 100% of errors
    
    - name: slow_requests
      condition: "duration > 5s"
      sample_rate: 1.0   # 100% of slow requests
    
    - name: deployments
      condition: "operation = 'deployment'"
      sample_rate: 1.0   # 100% of deployments
```

---

## 5. Alerting

### 5.1 Alert Categories

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Alert Severity                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  CRITICAL (P1) - Page immediately                               │
│  • Platform-wide outage                                         │
│  • Data loss risk                                               │
│  • Security breach                                              │
│                                                                  │
│  HIGH (P2) - Page during business hours                        │
│  • Agent error rate > 10%                                       │
│  • Deployment failures                                          │
│  • Database connectivity issues                                 │
│                                                                  │
│  MEDIUM (P3) - Ticket, next business day                       │
│  • Elevated latency                                             │
│  • Quota approaching limit                                      │
│  • Certificate expiring < 7 days                                │
│                                                                  │
│  LOW (P4) - Ticket, within sprint                              │
│  • Non-critical deprecation warnings                            │
│  • Performance optimization opportunities                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Alert Rules

```yaml
groups:
  - name: agent_alerts
    rules:
      # High error rate
      - alert: AgentHighErrorRate
        expr: agent:error_rate:5m > 0.1
        for: 5m
        labels:
          severity: high
        annotations:
          summary: "Agent {{ $labels.agent }} error rate > 10%"
          runbook: "https://docs.agentstack.io/runbooks/high-error-rate"
      
      # High latency
      - alert: AgentHighLatency
        expr: agent:latency_p99:5m > 5
        for: 10m
        labels:
          severity: medium
        annotations:
          summary: "Agent {{ $labels.agent }} P99 latency > 5s"
      
      # Cold start issues
      - alert: AgentColdStartSlow
        expr: |
          histogram_quantile(0.95, 
            sum(rate(knative_activator_request_latencies_bucket[5m])) by (le)
          ) > 3
        for: 15m
        labels:
          severity: medium
        annotations:
          summary: "Cold start P95 > 3s"
      
      # Token spend spike
      - alert: AgentTokenSpendSpike
        expr: |
          agent:tokens_per_hour > 1000000
          and
          agent:tokens_per_hour > 2 * agent:tokens_per_hour offset 1h
        for: 30m
        labels:
          severity: medium
        annotations:
          summary: "Token spend doubled for {{ $labels.agent }}"
```

### 5.3 Notification Channels

```yaml
alertmanager:
  routes:
    - receiver: pagerduty-critical
      match:
        severity: critical
    
    - receiver: slack-platform
      match:
        severity: high
    
    - receiver: email-team
      match:
        severity: medium
  
  receivers:
    - name: pagerduty-critical
      pagerduty_configs:
        - service_key: "${PD_SERVICE_KEY}"
    
    - name: slack-platform
      slack_configs:
        - channel: "#platform-alerts"
          send_resolved: true
```

---

## 6. Dashboards

### 6.1 Platform Overview Dashboard

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Platform Overview                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐         │
│  │ Active Agents │ │ Requests/min  │ │  Error Rate   │         │
│  │     1,234     │ │    45,678     │ │    0.02%      │         │
│  └───────────────┘ └───────────────┘ └───────────────┘         │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                Request Rate (24h)                        │    │
│  │  ████████████████████████████████████████████           │    │
│  │  ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄             │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌───────────────────────────┐ ┌───────────────────────────┐   │
│  │    Latency Distribution   │ │    Token Usage by Model   │   │
│  │    P50: 450ms             │ │    gpt-4o: 45%            │   │
│  │    P95: 1.2s              │ │    claude-3: 30%          │   │
│  │    P99: 2.8s              │ │    gemini: 25%            │   │
│  └───────────────────────────┘ └───────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Agent Detail Dashboard

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Agent: customer-support                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Status: ACTIVE    Replicas: 3    Revision: rev-042             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                Request Rate & Errors                     │    │
│  │  Requests ████████████████████████████                  │    │
│  │  Errors   ▁▁▁▂▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Tool Usage                             │    │
│  │  search-kb     ████████████████████  450 calls          │    │
│  │  create-ticket ████████              180 calls          │    │
│  │  lookup-order  ████                   90 calls          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Recent Logs:                                                    │
│  10:30:01 INFO  Request completed [200] 1.2s                    │
│  10:30:00 INFO  Tool call: search-kb 200ms                      │
│  10:29:58 WARN  Retry: LLM timeout, attempt 2                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Cost Tracking

### 7.1 Cost Metrics

```yaml
# Cost tracking metrics
cost_metrics:
  - name: agent_cost_tokens
    formula: (tokens_input * input_price) + (tokens_output * output_price)
    dimensions: [agent, model, project]
  
  - name: agent_cost_compute
    formula: pod_hours * instance_price
    dimensions: [agent, project]
  
  - name: agent_cost_total
    formula: cost_tokens + cost_compute + cost_storage
    dimensions: [project]
```

### 7.2 Cost Dashboard

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Cost Overview (MTD)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Total: $4,567.89                                               │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Cost by Category                            │    │
│  │  LLM Tokens   ████████████████████████  $3,200 (70%)    │    │
│  │  Compute      ████████                   $900 (20%)     │    │
│  │  Storage      ████                       $467 (10%)     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Top 5 Agents by Cost:                                          │
│  1. customer-support     $1,234                                 │
│  2. code-assistant       $987                                   │
│  3. data-analyzer        $654                                   │
│  4. onboarding-bot       $432                                   │
│  5. qa-agent             $321                                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. Implementation Checklist

### Phase 1: Core Observability
- [ ] Deploy OpenTelemetry Collector
- [ ] Configure Prometheus metrics
- [ ] Set up Loki for logs
- [ ] Basic Grafana dashboards

### Phase 2: Tracing & Alerts
- [ ] Instrument all services with OTEL
- [ ] Deploy Tempo/Jaeger
- [ ] Configure Alertmanager
- [ ] Create runbooks

### Phase 3: Advanced
- [ ] SLO dashboards
- [ ] Cost tracking
- [ ] Anomaly detection
- [ ] Custom agent metrics

---

## 9. References

- [OpenTelemetry](https://opentelemetry.io/docs/)
- [Prometheus](https://prometheus.io/docs/)
- [Grafana Loki](https://grafana.com/docs/loki/)
- [Grafana Tempo](https://grafana.com/docs/tempo/)
- [Knative Metrics](https://knative.dev/docs/serving/observability/metrics/)

---

**Previous**: [006-security-governance.md](006-security-governance.md)  
**Next**: [008-deployment-operations.md](008-deployment-operations.md)
