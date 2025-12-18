# Gateway API for Ingress

Kubernetes Gateway API patterns for AgentStack routing.

## Gateway API Overview

Gateway API is the evolution of Ingress, providing:
- Multi-tenant routing
- Cross-namespace references
- Traffic splitting
- Header-based routing

## Gateway Definition

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: agentstack-gateway
  namespace: agentstack-system
spec:
  gatewayClassName: contour
  listeners:
    - name: http
      port: 80
      protocol: HTTP
      hostname: "*.agentstack.io"
      allowedRoutes:
        namespaces:
          from: Selector
          selector:
            matchLabels:
              agentstack.io/gateway-access: "true"
    - name: https
      port: 443
      protocol: HTTPS
      hostname: "*.agentstack.io"
      tls:
        mode: Terminate
        certificateRefs:
          - name: agentstack-tls
            kind: Secret
      allowedRoutes:
        namespaces:
          from: Selector
          selector:
            matchLabels:
              agentstack.io/gateway-access: "true"
```

## HTTPRoute for API

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-routes
  namespace: agentstack
spec:
  parentRefs:
    - name: agentstack-gateway
      namespace: agentstack-system
  hostnames:
    - "api.agentstack.io"
  rules:
    # Health endpoints (no auth)
    - matches:
        - path:
            type: Exact
            value: /health
        - path:
            type: Exact
            value: /ready
      backendRefs:
        - name: agentstack-api
          port: 80
    
    # API v1 routes
    - matches:
        - path:
            type: PathPrefix
            value: /v1
      filters:
        - type: RequestHeaderModifier
          requestHeaderModifier:
            add:
              - name: X-Forwarded-Proto
                value: https
      backendRefs:
        - name: agentstack-api
          port: 80
```

## Traffic Splitting (Canary)

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-canary
  namespace: agentstack
spec:
  parentRefs:
    - name: agentstack-gateway
      namespace: agentstack-system
  hostnames:
    - "api.agentstack.io"
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /v1
      backendRefs:
        - name: agentstack-api
          port: 80
          weight: 90
        - name: agentstack-api-canary
          port: 80
          weight: 10
```

## Header-Based Routing

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-header-routing
  namespace: agentstack
spec:
  parentRefs:
    - name: agentstack-gateway
      namespace: agentstack-system
  rules:
    # Route beta users to canary
    - matches:
        - headers:
            - name: X-Beta-User
              value: "true"
          path:
            type: PathPrefix
            value: /v1
      backendRefs:
        - name: agentstack-api-canary
          port: 80
    
    # Default route
    - matches:
        - path:
            type: PathPrefix
            value: /v1
      backendRefs:
        - name: agentstack-api
          port: 80
```

## Rate Limiting with BackendPolicy

```yaml
apiVersion: projectcontour.io/v1alpha1
kind: ContourPolicy
metadata:
  name: api-rate-limit
  namespace: agentstack
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: api-routes
  rateLimit:
    global:
      descriptors:
        - entries:
            - requestHeader:
                name: X-Project-ID
                descriptorKey: project_id
      rateLimitService:
        name: ratelimit
        namespace: agentstack-system
        port: 8081
```

## Timeout Configuration

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-timeouts
  namespace: agentstack
spec:
  parentRefs:
    - name: agentstack-gateway
      namespace: agentstack-system
  rules:
    # Streaming endpoints need longer timeouts
    - matches:
        - path:
            type: PathPrefix
            value: /v1/agents
          path:
            type: RegularExpression
            value: ".*/chat/stream"
      backendRefs:
        - name: agentstack-api
          port: 80
      timeouts:
        request: 300s  # 5 minutes for streaming
        backendRequest: 300s
    
    # Normal API requests
    - matches:
        - path:
            type: PathPrefix
            value: /v1
      backendRefs:
        - name: agentstack-api
          port: 80
      timeouts:
        request: 30s
        backendRequest: 30s
```

## TLS Certificate (cert-manager)

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: agentstack-tls
  namespace: agentstack-system
spec:
  secretName: agentstack-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  commonName: "*.agentstack.io"
  dnsNames:
    - "*.agentstack.io"
    - "agentstack.io"
```

## GatewayClass (Contour)

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: contour
spec:
  controllerName: projectcontour.io/gateway-controller
  parametersRef:
    group: projectcontour.io
    kind: ContourDeployment
    name: contour-params
    namespace: projectcontour
```
