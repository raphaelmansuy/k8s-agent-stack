# Prometheus Alerting Rules for AgentStack

## API Service Alerts

```yaml
groups:
  - name: agentstack-api
    interval: 30s
    rules:
      # High Error Rate
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m])) 
          / sum(rate(http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | printf \"%.2f\" }}% (threshold: 5%)"
          runbook_url: "https://runbooks.agentstack.io/api/high-error-rate"

      # High Latency
      - alert: HighLatencyP95
        expr: |
          histogram_quantile(0.95, 
            sum(rate(http_request_duration_seconds_bucket[5m])) by (le)
          ) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High P95 latency"
          description: "P95 latency is {{ $value | printf \"%.2f\" }}s (threshold: 2s)"

      # High Latency Critical
      - alert: HighLatencyP99
        expr: |
          histogram_quantile(0.99, 
            sum(rate(http_request_duration_seconds_bucket[5m])) by (le)
          ) > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Critical P99 latency"
          description: "P99 latency is {{ $value | printf \"%.2f\" }}s (threshold: 5s)"

      # Low Request Rate (possible outage)
      - alert: LowRequestRate
        expr: sum(rate(http_requests_total[5m])) < 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Unusually low request rate"
          description: "Request rate is {{ $value | printf \"%.2f\" }} req/s"
```

## Agent Alerts

```yaml
groups:
  - name: agentstack-agents
    rules:
      # Agent Deployment Failure
      - alert: AgentDeploymentFailed
        expr: |
          increase(agent_deployment_failures_total[15m]) > 0
        labels:
          severity: warning
        annotations:
          summary: "Agent deployment failed"
          description: "{{ $labels.agent_id }} failed to deploy"

      # Agent High Error Rate
      - alert: AgentHighErrorRate
        expr: |
          sum by (agent_id) (rate(agent_requests_total{status="error"}[5m])) 
          / sum by (agent_id) (rate(agent_requests_total[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Agent {{ $labels.agent_id }} has high error rate"
          description: "Error rate: {{ $value | printf \"%.2f\" }}%"

      # Agent Cold Start Time
      - alert: AgentSlowColdStart
        expr: |
          histogram_quantile(0.95, 
            sum by (agent_id, le) (rate(agent_cold_start_duration_seconds_bucket[15m]))
          ) > 10
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Agent {{ $labels.agent_id }} has slow cold starts"
          description: "P95 cold start: {{ $value | printf \"%.2f\" }}s"
```

## LLM Service Alerts

```yaml
groups:
  - name: agentstack-llm
    rules:
      # LLM API Errors
      - alert: LLMAPIErrors
        expr: |
          sum by (provider) (rate(llm_calls_total{status="error"}[5m])) 
          / sum by (provider) (rate(llm_calls_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "LLM provider {{ $labels.provider }} errors"
          description: "Error rate: {{ $value | printf \"%.2f\" }}%"

      # LLM High Latency
      - alert: LLMHighLatency
        expr: |
          histogram_quantile(0.95, 
            sum by (provider, model) (rate(llm_call_duration_seconds_bucket[5m]))
          ) > 30
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "LLM {{ $labels.provider }}/{{ $labels.model }} slow"
          description: "P95 latency: {{ $value | printf \"%.2f\" }}s"

      # Token Usage Spike
      - alert: TokenUsageSpike
        expr: |
          sum(increase(chat_tokens_total[1h])) 
          > sum(increase(chat_tokens_total[1h] offset 1h)) * 2
        labels:
          severity: info
        annotations:
          summary: "Token usage doubled in last hour"
          description: "Current: {{ $value }} tokens"
```

## Database Alerts

```yaml
groups:
  - name: agentstack-database
    rules:
      # Connection Pool Exhaustion
      - alert: DBConnectionPoolExhausted
        expr: |
          pg_pool_connections_used / pg_pool_connections_max > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Database connection pool nearly exhausted"
          description: "Pool usage: {{ $value | printf \"%.2f\" }}%"

      # Slow Queries
      - alert: DBSlowQueries
        expr: |
          histogram_quantile(0.95, 
            sum(rate(pg_query_duration_seconds_bucket[5m])) by (le)
          ) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Database queries are slow"
          description: "P95 query time: {{ $value | printf \"%.2f\" }}s"

      # High Error Rate
      - alert: DBHighErrorRate
        expr: |
          sum(rate(pg_queries_total{status="error"}[5m])) 
          / sum(rate(pg_queries_total[5m])) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High database error rate"
          description: "Error rate: {{ $value | printf \"%.2f\" }}%"
```

## Infrastructure Alerts

```yaml
groups:
  - name: agentstack-infrastructure
    rules:
      # Pod Restarts
      - alert: PodCrashLooping
        expr: |
          increase(kube_pod_container_status_restarts_total{namespace="agentstack"}[1h]) > 3
        labels:
          severity: warning
        annotations:
          summary: "Pod {{ $labels.pod }} is crash looping"
          description: "{{ $value }} restarts in the last hour"

      # CPU Throttling
      - alert: CPUThrottling
        expr: |
          sum by (pod) (rate(container_cpu_cfs_throttled_seconds_total{namespace="agentstack"}[5m])) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Pod {{ $labels.pod }} is being CPU throttled"

      # Memory Pressure
      - alert: HighMemoryUsage
        expr: |
          container_memory_working_set_bytes{namespace="agentstack"} 
          / container_spec_memory_limit_bytes{namespace="agentstack"} > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Pod {{ $labels.pod }} high memory usage"
          description: "Memory usage: {{ $value | printf \"%.2f\" }}%"

      # Disk Pressure
      - alert: PVCNearFull
        expr: |
          kubelet_volume_stats_used_bytes{namespace="agentstack"} 
          / kubelet_volume_stats_capacity_bytes{namespace="agentstack"} > 0.85
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "PVC {{ $labels.persistentvolumeclaim }} nearly full"
          description: "Usage: {{ $value | printf \"%.2f\" }}%"
```

## SLO-Based Alerts

```yaml
groups:
  - name: agentstack-slos
    rules:
      # Availability SLO (99.9%)
      - alert: AvailabilitySLOBreach
        expr: |
          (1 - (
            sum(rate(http_requests_total{status=~"5.."}[30d])) 
            / sum(rate(http_requests_total[30d]))
          )) < 0.999
        labels:
          severity: critical
          slo: availability
        annotations:
          summary: "Availability SLO at risk"
          description: "Current availability: {{ $value | printf \"%.4f\" }}"

      # Latency SLO (P95 < 500ms)
      - alert: LatencySLOBreach
        expr: |
          histogram_quantile(0.95, 
            sum(rate(http_request_duration_seconds_bucket[30d])) by (le)
          ) > 0.5
        labels:
          severity: critical
          slo: latency
        annotations:
          summary: "Latency SLO at risk"
          description: "P95 latency: {{ $value | printf \"%.3f\" }}s"

      # Error Budget Burn Rate
      - alert: ErrorBudgetFastBurn
        expr: |
          (
            sum(rate(http_requests_total{status=~"5.."}[1h])) 
            / sum(rate(http_requests_total[1h]))
          ) > (1 - 0.999) * 14.4
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Error budget burning too fast"
          description: "At this rate, monthly error budget exhausted in < 2 days"
```

## Alert Routing (Alertmanager)

```yaml
# alertmanager.yml
route:
  receiver: default
  group_by: [alertname, severity]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - match:
        severity: critical
      receiver: pagerduty
      continue: true
    - match:
        severity: warning
      receiver: slack-warnings
    - match_re:
        slo: ".*"
      receiver: slo-channel

receivers:
  - name: default
    webhook_configs:
      - url: http://webhook/default

  - name: pagerduty
    pagerduty_configs:
      - service_key: ${PAGERDUTY_KEY}

  - name: slack-warnings
    slack_configs:
      - api_url: ${SLACK_WEBHOOK}
        channel: '#agentstack-alerts'
        
  - name: slo-channel
    slack_configs:
      - api_url: ${SLACK_WEBHOOK}
        channel: '#agentstack-slo'
```
