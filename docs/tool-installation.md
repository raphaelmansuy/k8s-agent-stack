# Tool Installation Guide

Installation instructions for required and optional tools.

---

## Required Tools

### kubectl

Kubernetes command-line tool for cluster management.

**macOS:**

```bash
brew install kubectl
```

**Linux:**

```bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

**Verify:**

```bash
kubectl version --client
```

📖 [Official docs](https://kubernetes.io/docs/tasks/tools/)

---

### kn CLI (Knative)

Knative command-line interface for managing Knative services.

**macOS:**

```bash
brew install knative/client/kn
```

**Linux:**

```bash
curl -L https://github.com/knative/client/releases/download/knative-v1.20.0/kn-linux-amd64 \
  -o /usr/local/bin/kn
chmod +x /usr/local/bin/kn
```

**Verify:**

```bash
kn version
```

📖 [Official docs](https://knative.dev/docs/client/install-kn/)

---

### Docker

Container runtime for building and running containers.

**macOS - Docker Desktop:**

```bash
brew install --cask docker
```

**macOS - OrbStack (recommended for M1/M2):**

```bash
brew install --cask orbstack
```

**Linux:**

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
```

**Verify:**

```bash
docker --version
```

📖 Official docs: [Docker](https://docs.docker.com/get-docker/) | [OrbStack](https://orbstack.dev/)

---

## Optional Tools

### Helm

Kubernetes package manager for installing charts.

**macOS:**

```bash
brew install helm
```

**Linux:**

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

---

## Go Development Tools (SOTA 2025)

For developers working on the `agentstack` core or Go-based agents, we recommend the following toolset.

### Go 1.24+

**macOS:**

```bash
brew install go
```

**Linux:**

```bash
# Follow instructions at https://go.dev/doc/install
```

### Automated Tool Installation

The project includes a `Makefile` target to install all necessary development tools (linters, formatters, security scanners).

```bash
# From the project root
make install-tools
```

This installs:

- **golangci-lint**: High-performance Go linter.
- **govulncheck**: Real-time vulnerability scanning.
- **gofumpt**: Stricter, more idiomatic Go formatter.
- **sqlc**: Type-safe SQL generator.
- **atlas**: Modern database migrations.
- **air**: Live reload for Go apps.
- **mockgen**: Mocking framework for testing.

---

## VS Code Optimization

We provide a pre-configured SOTA (State of the Art) VS Code setup in `.vscode/settings.json`.

**Key Features:**

- **Security**: Real-time vulnerability scanning via `govulncheck`.
- **Strictness**: Automatic formatting with `gofumpt` and strict linting.
- **Performance**: Optimized `gopls` settings for large monorepos.
- **AI Ready**: Optimized for GitHub Copilot or Cursor.

**Verify:**

```bash
helm version
```

📖 [Official docs](https://helm.sh/docs/intro/install/)

---

### k9s

Terminal UI for Kubernetes cluster management.

**macOS:**

```bash
brew install derailed/k9s/k9s
```

**Linux:**

```bash
curl -sS https://webinstall.dev/k9s | bash
```

📖 [Official docs](https://k9scli.io/)

---

### stern

Multi-pod log tailing for Kubernetes.

**macOS:**

```bash
brew install stern
```

**Linux:**

```bash
go install github.com/stern/stern@latest
```

**Usage:**

```bash
stern -n kagent google-adk-agent
```

📖 [Official docs](https://github.com/stern/stern)

---

## Quick Reference

| Tool      | Install (macOS)                  | Purpose           |
| --------- | -------------------------------- | ----------------- |
| `kubectl` | `brew install kubectl`           | Kubernetes CLI    |
| `kn`      | `brew install knative/client/kn` | Knative CLI       |
| `docker`  | `brew install --cask docker`     | Container runtime |
| `helm`    | `brew install helm`              | Package manager   |
| `k9s`     | `brew install derailed/k9s/k9s`  | Terminal UI       |
| `stern`   | `brew install stern`             | Log tailing       |

---

## Verify All Tools

```bash
# Check all required tools
kubectl version --client && \
kn version && \
docker --version && \
echo "✅ All required tools installed"
```

---

## Next Steps

- [Getting Started](getting-started.md) - Install the platform
- [Deployment Guide](deployment-guide.md) - Deploy agents

---

[← Back to Documentation Index](README.md) • [Getting Started](getting-started.md) • [Architecture](architecture.md) • [Main README](../README.md)
