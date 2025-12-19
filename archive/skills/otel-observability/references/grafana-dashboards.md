# Grafana Dashboards for AgentStack

## Overview Dashboard

```json
{
  "title": "AgentStack Overview",
  "panels": [
    {
      "title": "Request Rate",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(rate(http_requests_total[5m]))",
          "legendFormat": "Requests/s"
        }
      ]
    },
    {
      "title": "Error Rate",
      "type": "gauge",
      "targets": [
        {
          "expr": "sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m])) * 100",
          "legendFormat": "Error %"
        }
      ],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          {"value": 0, "color": "green"},
          {"value": 1, "color": "yellow"},
          {"value": 5, "color": "red"}
        ]
      }
    },
    {
      "title": "P95 Latency",
      "type": "stat",
      "targets": [
        {
          "expr": "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))",
          "legendFormat": "P95"
        }
      ],
      "unit": "s"
    },
    {
      "title": "Active Agents",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(agents_active)",
          "legendFormat": "Active"
        }
      ]
    }
  ]
}
```

## Agent Performance Dashboard

### Query Templates

```promql
# Agent request rate by framework
sum by (framework) (rate(agent_requests_total[5m]))

# Agent latency by agent
histogram_quantile(0.95, 
  sum by (agent_id, le) (rate(agent_request_duration_seconds_bucket[5m]))
)

# Agent error rate
sum by (agent_id) (rate(agent_requests_total{status="error"}[5m])) 
/ sum by (agent_id) (rate(agent_requests_total[5m])) * 100

# Token usage by project
sum by (project_id) (increase(chat_tokens_total[1h]))

# LLM latency percentiles
histogram_quantile(0.50, sum(rate(llm_call_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.95, sum(rate(llm_call_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.99, sum(rate(llm_call_duration_seconds_bucket[5m])) by (le))
```

## LLM Usage Dashboard

### Token Tracking

```promql
# Total tokens by model
sum by (model) (increase(llm_tokens_input_total[24h]))
sum by (model) (increase(llm_tokens_output_total[24h]))

# Cost estimation (example rates)
(sum(increase(llm_tokens_input_total{model="gpt-4"}[24h])) * 0.00003) +
(sum(increase(llm_tokens_output_total{model="gpt-4"}[24h])) * 0.00006)

# Tokens per conversation
sum by (session_id) (increase(chat_tokens_total[1h]))
```

## Distributed Tracing Integration

### Tempo Data Source Queries

```promql
# Trace count by service
sum by (service_name) (rate(traces_spanmetrics_calls_total[5m]))

# Trace latency by operation
histogram_quantile(0.95,
  sum by (service_name, span_name, le) (
    rate(traces_spanmetrics_latency_bucket[5m])
  )
)

# Error traces
sum by (service_name, span_name) (
  rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR"}[5m])
)
```

## Dashboard Variables

```yaml
# Common variables
- name: project
  type: query
  query: label_values(http_requests_total, project_id)

- name: agent
  type: query  
  query: label_values(agent_requests_total{project_id="$project"}, agent_id)

- name: environment
  type: query
  query: label_values(http_requests_total, environment)

- name: interval
  type: interval
  options:
    - 1m
    - 5m
    - 15m
    - 1h
```

## RED Method Panels

### Rate
```promql
sum(rate(http_requests_total{project_id="$project"}[$interval]))
```

### Error
```promql
sum(rate(http_requests_total{project_id="$project", status=~"5.."}[$interval])) 
/ sum(rate(http_requests_total{project_id="$project"}[$interval])) * 100
```

### Duration
```promql
histogram_quantile(0.95, 
  sum by (le) (rate(http_request_duration_seconds_bucket{project_id="$project"}[$interval]))
)
```

## USE Method Panels (for infrastructure)

### Utilization
```promql
# CPU utilization
avg by (pod) (rate(container_cpu_usage_seconds_total{namespace="agentstack"}[5m]))
/ avg by (pod) (container_spec_cpu_quota{namespace="agentstack"} / 100000) * 100

# Memory utilization
avg by (pod) (container_memory_working_set_bytes{namespace="agentstack"})
/ avg by (pod) (container_spec_memory_limit_bytes{namespace="agentstack"}) * 100
```

### Saturation
```promql
# CPU throttling
sum by (pod) (rate(container_cpu_cfs_throttled_seconds_total{namespace="agentstack"}[5m]))
```

### Errors
```promql
# Container restarts
sum by (pod) (increase(kube_pod_container_status_restarts_total{namespace="agentstack"}[1h]))
```
