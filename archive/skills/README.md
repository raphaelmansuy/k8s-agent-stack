# AgentStack Skills Library

> Claude Skills for building the Sovereign GenAI Agent Platform

## Overview

This directory contains **11 specialized Claude Skills** designed to ensure 100% success when building the AgentStack platform. Each skill follows the [Anthropic Skills format](https://github.com/anthropics/skills) with SKILL.md files and bundled resources.

## Skills Index

| Skill | Description | Triggers |
|-------|-------------|----------|
| **[skill-creator](./skill-creator/)** | Meta-skill for creating new skills | "create skill", "new skill" |
| **[go-api-gateway](./go-api-gateway/)** | REST API implementation with Go Fiber | "API", "handler", "endpoint", "SSE" |
| **[kubernetes-manifests](./kubernetes-manifests/)** | K8s resource generation | "deployment", "service", "manifest" |
| **[knative-serving](./knative-serving/)** | Serverless agent runtime | "Knative", "autoscaling", "cold start" |
| **[agent-evaluation-mlflow](./agent-evaluation-mlflow/)** | Safety and quality evaluation | "MLflow", "evaluation", "scorer", "safety" |
| **[a2a-protocol-impl](./a2a-protocol-impl/)** | Agent-to-Agent communication | "A2A", "agent communication", "multi-agent" |
| **[multi-tenant-postgres](./multi-tenant-postgres/)** | Database with tenant isolation | "PostgreSQL", "RLS", "multi-tenant", "database" |
| **[otel-observability](./otel-observability/)** | OpenTelemetry instrumentation | "OTEL", "tracing", "metrics", "Prometheus" |
| **[agentctl-cli](./agentctl-cli/)** | CLI tool development | "CLI", "agentctl", "cobra" |
| **[agent-deployment-pipeline](./agent-deployment-pipeline/)** | CI/CD with evaluation gates | "pipeline", "CI/CD", "GitOps", "deploy" |
| **[security-rbac-auth](./security-rbac-auth/)** | Authentication & authorization | "JWT", "RBAC", "API key", "OAuth" |

## Skill Architecture

```
skills/
├── skill-creator/              # Meta-skill for creating skills
│   ├── SKILL.md
│   └── references/
│       ├── skill-patterns.md
│       └── agentstack-context.md
│
├── go-api-gateway/             # Go REST API with Fiber
│   ├── SKILL.md
│   ├── references/
│   │   ├── middleware-patterns.md
│   │   └── sse-streaming.md
│   └── assets/
│       └── handler-template.go
│
├── kubernetes-manifests/       # K8s resource generation
│   ├── SKILL.md
│   └── references/
│       └── gateway-api.md
│
├── knative-serving/            # Serverless runtime
│   ├── SKILL.md
│   └── references/
│       ├── cold-start-optimization.md
│       └── kagent-integration.md
│
├── agent-evaluation-mlflow/    # MLflow evaluation
│   ├── SKILL.md
│   └── references/
│       └── scorer-catalog.md
│
├── a2a-protocol-impl/          # Agent-to-Agent protocol
│   └── SKILL.md
│
├── multi-tenant-postgres/      # PostgreSQL multi-tenancy
│   └── SKILL.md
│
├── otel-observability/         # OpenTelemetry
│   ├── SKILL.md
│   └── references/
│       ├── grafana-dashboards.md
│       └── alerting-rules.md
│
├── agentctl-cli/               # CLI development
│   ├── SKILL.md
│   └── references/
│       └── cobra-patterns.md
│
├── agent-deployment-pipeline/  # CI/CD pipelines
│   └── SKILL.md
│
├── security-rbac-auth/         # Security & Auth
│   └── SKILL.md
│
└── scripts/
    └── init_skill.py           # Skill scaffolding script
```

## How to Use These Skills

### With Claude Projects

1. Create a new Claude Project
2. Add skill directories to Project Knowledge
3. Claude will automatically use relevant skills based on your prompts

### Skill Activation

Skills activate automatically based on trigger phrases in your prompts. Examples:

```
"Create a new API endpoint for agents"
→ Activates: go-api-gateway, kubernetes-manifests

"Deploy the agent to production with safety gates"
→ Activates: agent-deployment-pipeline, agent-evaluation-mlflow, knative-serving

"Implement multi-tenant database schema"
→ Activates: multi-tenant-postgres

"Add OpenTelemetry tracing"
→ Activates: otel-observability
```

### Creating New Skills

Use the meta-skill or the init script:

```bash
# Using the script
python skills/scripts/init_skill.py my-new-skill

# Creates:
# skills/my-new-skill/
# ├── SKILL.md
# ├── references/
# ├── scripts/
# └── assets/
```

## Skill Format

Each skill follows the Anthropic SKILL.md format:

```yaml
---
name: skill-name
description: What this skill does and when to activate it. Include trigger words.
---

# Skill Title

## Overview
Brief description of the skill's purpose.

## Core Content
Main instructions, patterns, and code examples.

## Resources
References to bundled files in references/ and assets/
```

## Platform Coverage Matrix

| Layer | Skills Covering |
|-------|-----------------|
| **Infrastructure** | kubernetes-manifests, otel-observability |
| **Runtime** | knative-serving, a2a-protocol-impl |
| **Data** | multi-tenant-postgres |
| **API** | go-api-gateway, security-rbac-auth |
| **Evaluation** | agent-evaluation-mlflow |
| **DevEx** | agentctl-cli, agent-deployment-pipeline |
| **Meta** | skill-creator |

## Spec Document Mapping

| Spec | Primary Skills |
|------|----------------|
| `spec/001-platform-overview.md` | skill-creator (context) |
| `spec/002-architecture-layers.md` | All skills |
| `spec/004-api-design.md` | go-api-gateway |
| `spec/005-knative-integration.md` | knative-serving |
| `spec/006-security-governance.md` | security-rbac-auth, agent-evaluation-mlflow |
| `spec/007-observability.md` | otel-observability |
| `spec/009-developer-experience.md` | agentctl-cli |
| `spec/010-agent-evaluation.md` | agent-evaluation-mlflow |

## Contributing

When adding or updating skills:

1. Follow the SKILL.md format with YAML frontmatter
2. Include clear trigger words in the description
3. Add references/ for supplementary documentation
4. Add assets/ for code templates
5. Update this README with the new skill

## License

Apache License 2.0 - Same as AgentStack platform.
