# Cold Start Optimization

Strategies to minimize Knative cold start latency for AI agents.

## Understanding Cold Start

Cold start occurs when:
1. No pods exist (scaled to zero)
2. Request arrives
3. Pod must be scheduled, started, and become ready

Typical cold start: 5-30 seconds for AI agents.

## Optimization Strategies

### 1. Minimum Scale

Keep at least one pod running:

```yaml
annotations:
  autoscaling.knative.dev/min-scale: "1"
```

Trade-off: Cost vs latency. Use for production critical agents.

### 2. Initial Scale

Start with multiple pods on deploy:

```yaml
annotations:
  autoscaling.knative.dev/initial-scale: "2"
```

### 3. Scale-Down Delay

Wait before scaling down:

```yaml
annotations:
  autoscaling.knative.dev/scale-down-delay: "5m"
```

### 4. Optimize Container Image

```dockerfile
# Use smaller base images
FROM python:3.12-slim AS runtime

# Pre-download models during build
RUN python -c "from transformers import AutoTokenizer; AutoTokenizer.from_pretrained('gpt2')"

# Use multi-stage builds
FROM builder AS final
COPY --from=builder /app /app
```

### 5. Reduce Readiness Probe Time

```yaml
spec:
  containers:
    - readinessProbe:
        httpGet:
          path: /health
          port: 8080
        initialDelaySeconds: 1    # Start checking quickly
        periodSeconds: 1          # Check frequently
        successThreshold: 1       # One success is enough
        failureThreshold: 3
```

### 6. Lazy Loading

Load models only when needed:

```python
# agent.py
class Agent:
    def __init__(self):
        self._model = None  # Lazy load
    
    @property
    def model(self):
        if self._model is None:
            self._model = load_model()
        return self._model
```

### 7. Model Caching with Volumes

```yaml
spec:
  containers:
    - volumeMounts:
        - name: model-cache
          mountPath: /root/.cache/huggingface
  volumes:
    - name: model-cache
      persistentVolumeClaim:
        claimName: model-cache-pvc
```

### 8. Activator Buffer Size

Configure global activator:

```yaml
# config-autoscaler ConfigMap
data:
  activator-capacity: "500"
```

### 9. Pre-warm with Synthetic Traffic

```bash
# Send periodic requests to keep warm
*/5 * * * * curl -s https://agent.agentstack.io/health
```

## Cold Start Monitoring

### Metrics to Track

```promql
# Cold start rate
rate(revision_app_request_latencies_bucket{le="5000"}[5m])

# Activator queue depth
activator_request_concurrency

# Pod startup time
kube_pod_start_time - kube_pod_created
```

### Alerting

```yaml
groups:
  - name: knative-cold-start
    rules:
      - alert: HighColdStartRate
        expr: |
          sum(rate(activator_request_count[5m])) /
          sum(rate(revision_request_count[5m])) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High cold start rate detected"
```

## Cost vs Latency Matrix

| Strategy | Latency Impact | Cost Impact |
|----------|---------------|-------------|
| min-scale: 0 | High (cold start) | Lowest |
| min-scale: 1 | Low | Moderate |
| min-scale: 2+ | Minimal | Higher |
| scale-down-delay: 5m | Reduced | Slight increase |
| Pre-warm requests | Reduced | Minimal |

## Recommended Configuration by Tier

### Development

```yaml
annotations:
  autoscaling.knative.dev/min-scale: "0"
  autoscaling.knative.dev/max-scale: "5"
```

### Production (Standard)

```yaml
annotations:
  autoscaling.knative.dev/min-scale: "1"
  autoscaling.knative.dev/max-scale: "20"
  autoscaling.knative.dev/scale-down-delay: "2m"
```

### Production (Critical)

```yaml
annotations:
  autoscaling.knative.dev/min-scale: "2"
  autoscaling.knative.dev/max-scale: "50"
  autoscaling.knative.dev/scale-down-delay: "5m"
```
