---
name: kubernetes-manifests
description: Generate production-ready Kubernetes manifests for AgentStack. Use for creating Deployments, Services, ConfigMaps, Secrets, RBAC, and other K8s resources. Triggers on "create deployment", "k8s manifest", "kubernetes yaml", "pod spec", "service definition", "configmap", "RBAC", or when deploying components to Kubernetes.
---

# Kubernetes Manifests

## Overview

Generate production-ready Kubernetes manifests following AgentStack conventions and best practices for deploying on any K8s distribution.

## Directory Structure

```
deployments/
├── base/                    # Base manifests (kustomize)
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── api-gateway/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   ├── worker/
│   └── common/
│       ├── configmap.yaml
│       └── secrets.yaml
├── overlays/
│   ├── dev/
│   │   └── kustomization.yaml
│   ├── staging/
│   │   └── kustomization.yaml
│   └── prod/
│       └── kustomization.yaml
└── crds/                    # Custom Resource Definitions
```

## Core Manifest Patterns

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentstack-api
  namespace: agentstack
  labels:
    app.kubernetes.io/name: agentstack-api
    app.kubernetes.io/component: api
    app.kubernetes.io/part-of: agentstack
    app.kubernetes.io/version: "1.0.0"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app.kubernetes.io/name: agentstack-api
  template:
    metadata:
      labels:
        app.kubernetes.io/name: agentstack-api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: agentstack-api
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
        - name: api
          image: ghcr.io/raphaelmansuy/agentstack-api:v1.0.0
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
            - name: metrics
              containerPort: 9090
              protocol: TCP
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          envFrom:
            - configMapRef:
                name: agentstack-config
            - secretRef:
                name: agentstack-secrets
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 1000m
              memory: 1Gi
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app.kubernetes.io/name: agentstack-api
                topologyKey: kubernetes.io/hostname
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: agentstack-api
  namespace: agentstack
  labels:
    app.kubernetes.io/name: agentstack-api
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 80
      targetPort: http
      protocol: TCP
    - name: metrics
      port: 9090
      targetPort: metrics
      protocol: TCP
  selector:
    app.kubernetes.io/name: agentstack-api
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: agentstack-config
  namespace: agentstack
data:
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
  HTTP_PORT: "8080"
  METRICS_PORT: "9090"
  DATABASE_HOST: "postgres.agentstack.svc.cluster.local"
  DATABASE_PORT: "5432"
  DATABASE_NAME: "agentstack"
  REDIS_HOST: "redis.agentstack.svc.cluster.local"
  REDIS_PORT: "6379"
  OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector.observability:4317"
```

### Secret (with External Secrets Operator)

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: agentstack-secrets
  namespace: agentstack
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: agentstack-secrets
    creationPolicy: Owner
  data:
    - secretKey: DATABASE_PASSWORD
      remoteRef:
        key: agentstack/database
        property: password
    - secretKey: JWT_SECRET
      remoteRef:
        key: agentstack/auth
        property: jwt_secret
    - secretKey: REDIS_PASSWORD
      remoteRef:
        key: agentstack/redis
        property: password
```

### HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: agentstack-api
  namespace: agentstack
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: agentstack-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 4
          periodSeconds: 15
      selectPolicy: Max
```

### PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: agentstack-api
  namespace: agentstack
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app.kubernetes.io/name: agentstack-api
```

## RBAC Pattern

```yaml
# ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: agentstack-api
  namespace: agentstack
---
# Role for namespace-scoped permissions
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agentstack-api
  namespace: agentstack
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
---
# RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: agentstack-api
  namespace: agentstack
subjects:
  - kind: ServiceAccount
    name: agentstack-api
    namespace: agentstack
roleRef:
  kind: Role
  name: agentstack-api
  apiGroup: rbac.authorization.k8s.io
```

## Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: agentstack-api
  namespace: agentstack
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: agentstack-api
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: envoy
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: postgres
      ports:
        - protocol: TCP
          port: 5432
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: redis
      ports:
        - protocol: TCP
          port: 6379
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: UDP
          port: 53  # DNS
```

## Kustomization

```yaml
# deployments/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: agentstack

resources:
  - namespace.yaml
  - api-gateway/deployment.yaml
  - api-gateway/service.yaml
  - api-gateway/hpa.yaml
  - common/configmap.yaml

commonLabels:
  app.kubernetes.io/part-of: agentstack
  app.kubernetes.io/managed-by: kustomize
```

```yaml
# deployments/overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: agentstack-prod

resources:
  - ../../base

namePrefix: prod-

patches:
  - path: replicas-patch.yaml
  - path: resources-patch.yaml

configMapGenerator:
  - name: agentstack-config
    behavior: merge
    literals:
      - LOG_LEVEL=warn
```

## Resources

- `references/gateway-api.md` - Gateway API for ingress
- `references/security-hardening.md` - Security best practices
- `assets/` - Template manifests for common patterns
