# Makefile Quick Reference Guide

This guide shows the most common `make` commands for working with k8s-agent-stack.

## 🚀 Quick Start (Copy & Paste)

```bash
# Install tools (one time)
make install

# Setup local Kubernetes with Knative
make setup

# Deploy example agent
make deploy

# View agent logs
make agent-logs

# Port-forward to access agent locally
make port-forward

# Clean up
make clean
```

## 📋 Complete Command Reference

### Installation & Setup

```bash
make install            # Install required tools (kn, kubectl, helm, docker)
make setup              # Setup local K8s with Knative + kagent
make setup-metrics      # Setup with metrics server for autoscaling
make setup-warm         # Setup with warm pods (1 replica always running)
make verify             # Verify installation is complete
```

### Agent Deployment

```bash
make deploy             # Deploy example Google ADK agent
make deploy-k8s-helper  # Deploy Kubernetes helper agent
make undeploy           # Remove the agent
make list-agents        # List all deployed agents
make update-agent       # Update agent to new image version
```

### Testing & Building

```bash
make test               # Run all tests (unit + integration)
make test-unit          # Run unit tests only
make test-integration   # Run integration tests only
make test-coverage      # Run tests with coverage report (generates HTML)
make build              # Build Docker image (requires Docker registry configured)
make build-dev          # Build development Docker image
make push               # Push Docker image to registry
```

### Monitoring & Debugging

```bash
make agent-status       # Check deployment status
make agent-logs         # View logs in real-time (Ctrl+C to exit)
make agent-logs-previous # View logs from previous crashed pod
make agent-describe     # Show detailed pod information
make port-forward       # Port-forward agent to localhost:8080
make port-forward-custom # Port-forward with custom port
make test-agent         # Test agent endpoint (requires port-forward running)
make watch-pods         # Watch pod scaling live
make knative-logs       # View Knative autoscaler logs
make shell              # Open bash shell in agent pod
```

### Cleanup & Maintenance

```bash
make clean              # Remove all agents
make clean-all          # REMOVE EVERYTHING (Knative, kagent, all agents)
make debug              # Run full diagnostics
```

### Development & Production

```bash
make dev                # Setup complete development environment
make dev-watch          # Auto-rebuild agent when code changes
make prod               # Setup production environment
```

### Documentation & Info

```bash
make help               # Show this help
make docs               # List available documentation
make version            # Show component versions
make env                # Show environment configuration
make traffic-split      # Show canary deployment example
make scale-config       # Show current scaling settings
```

## 🎯 Common Workflows

### Local Development Workflow

```bash
# First time setup
make install
make setup
make deploy

# In another terminal
make port-forward

# Test your agent
make test-agent

# View logs as you test
make agent-logs

# When done
make clean
```

### Building & Deploying Your Own Agent

```bash
# Configure Docker registry (edit Makefile or set env var)
export DOCKER_REGISTRY=gcr.io/my-project

# Build your agent
make build

# Push to registry
make push

# Deploy
make deploy

# Monitor
make agent-logs
make watch-pods
```

### Production Setup

```bash
# Setup production environment
make prod

# Deploy agent
make deploy

# Monitor with metrics
make watch-pods
make knative-logs
```

### Testing

```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration

# Generate coverage report (opens HTML in htmlcov/)
make test-coverage
```

## 🔧 Configuring the Makefile

Edit these variables in the Makefile to customize:

```makefile
PROJECT_NAME := k8s-agent-stack      # Project name
AGENT_NAME := google-adk-agent       # Agent name
AGENT_NAMESPACE := kagent            # Kubernetes namespace
DOCKER_REGISTRY := gcr.io/your-project  # Docker registry
AGENT_VERSION := v1.0.0              # Agent version
```

Or set environment variables:

```bash
export DOCKER_REGISTRY=docker.io/myusername
export AGENT_VERSION=v2.0.0
make build push
```

## 📊 Output & Colors

The Makefile uses colors for clarity:
- 🔵 Blue: Section headers and informational messages
- 🟢 Green: Success messages (✓ Complete)
- 🟡 Yellow: Warnings and ongoing actions
- 🔴 Red: Errors and dangerous operations

## ⚡ Pro Tips

### 1. Chain commands together
```bash
make setup && make deploy && make port-forward
```

### 2. View agent logs in real-time while testing
```bash
# Terminal 1
make agent-logs

# Terminal 2
make port-forward

# Terminal 3
make test-agent
```

### 3. Watch pod scaling
```bash
make watch-pods
```

### 4. Debug pod issues
```bash
# Check pod details
make agent-describe

# View previous crash logs
make agent-logs-previous

# Open shell in pod
make shell

# Run full diagnostics
make debug
```

### 5. Canary deployment
```bash
# Deploy new version
make build && make push && make deploy

# Check current scaling
make scale-config

# View traffic splitting example
make traffic-split
```

## 🐛 Troubleshooting

### "make: kubectl not found"
```bash
make install
# Or manually install: brew install kubectl
```

### "make setup" fails
```bash
# Verify Kubernetes is running
make verify

# Run diagnostics
make debug
```

### Agent not responding
```bash
# Check pod status
make agent-status

# View logs
make agent-logs

# Check pod details
make agent-describe
```

### Port-forward issues
```bash
# Kill existing port-forward
pkill -f "port-forward.*8080"

# Try again
make port-forward
```

## 📚 Learn More

- Full documentation: `make docs`
- View component versions: `make version`
- Development setup: `make dev`
- Production setup: `make prod`

## ❓ Need Help?

```bash
# Show all available commands
make help

# Show detailed help for specific task
make <target> # Will show what it does

# Check configuration
make env

# Run diagnostics
make debug
```

---

**Last Updated**: December 2025
**License**: Apache 2.0
**Author**: Raphaël MANSUY
