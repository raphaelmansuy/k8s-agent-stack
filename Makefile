# k8s-agent-stack Makefile (Simplified)
# Copyright 2025 Raphaël MANSUY
# Licensed under the Apache License, Version 2.0
#
# This Makefile uses the official kagent CLI for installation.
# Documentation: https://kagent.dev/docs

.PHONY: help start install setup ui status clean uninstall

# Variables
AGENT_NAMESPACE := kagent
KAGENT_PROFILE := demo

# Colors (macOS compatible - using printf)
define print_header
	@printf '\033[0;34m╔════════════════════════════════════════════════════════════════╗\033[0m\n'
	@printf '\033[0;34m║         k8s-agent-stack - AI Agent Platform                    ║\033[0m\n'
	@printf '\033[0;34m╚════════════════════════════════════════════════════════════════╝\033[0m\n'
endef

# Default target
.DEFAULT_GOAL := help

help: ## Show this help
	$(print_header)
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  make start         Complete setup (one command!)"
	@echo "  make ui            Open Kagent web interface (keep terminal open)"
	@echo "  make status        Check everything is running"
	@echo ""
	@echo "🤖 ADK Agent:"
	@echo "  make adk-agent     Build and deploy Google ADK agent"
	@echo ""
	@echo "📋 Commands:"
	@echo "  make install       Install kagent CLI"
	@echo "  make setup         Install kagent to cluster"
	@echo "  make clean         Remove kagent"
	@echo ""
	@echo "💡 Tip: 'make ui' must stay running to access the web interface"
	@echo ""
	@echo "📖 More:"
	@echo "  make help-all      Show all commands"
	@echo ""

help-all: ## Show all commands
	$(print_header)
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1;33m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""

##@ Quick Start

start: check-cluster check-api-key install-cli setup-kagent configure-api-key ## 🚀 Complete setup in one command
	@echo ""
	@printf '\033[0;32m✓ Setup complete!\033[0m\n'
	@echo ""
	@echo "Next steps:"
	@echo "  make ui     → Open Kagent web interface"
	@echo "  make status → Check component status"
	@echo "  make agents → List all agents"
	@echo ""

ui: ## Open Kagent UI (http://localhost:8080)
	@printf '\033[0;32m╔════════════════════════════════════════╗\033[0m\n'
	@printf '\033[0;32m║  Kagent UI → http://localhost:8080    ║\033[0m\n'
	@printf '\033[0;32m╚════════════════════════════════════════╝\033[0m\n'
	@echo ""
	@echo "🌐 Starting port-forward..."
	@echo "   Keep this terminal open while using the UI"
	@echo "   Press Ctrl+C to stop"
	@echo ""
	@kubectl port-forward -n $(AGENT_NAMESPACE) svc/kagent-ui 8080:8080

dashboard: ui ## Alias for ui

ui-port-forward: ui ## Alias for ui

status: ## Check cluster and kagent status
	@echo "📊 Cluster Status"
	@echo "────────────────"
	@kubectl cluster-info 2>/dev/null | head -1 || echo "  ✗ Not connected to cluster"
	@echo ""
	@echo "📦 Kagent Components"
	@echo "────────────────────"
	@kubectl get pods -n $(AGENT_NAMESPACE) 2>/dev/null || echo "  ✗ Kagent not installed"
	@echo ""
	@echo "🤖 Agents"
	@echo "─────────"
	@kubectl get agents -n $(AGENT_NAMESPACE) 2>/dev/null || echo "  No agents found"
	@echo ""

##@ AgentStack API

agentstack-build: ## Build AgentStack API and Worker images
	@echo "Building AgentStack images..."
	cd agentstack && make docker-build
	docker tag agentstack-api:latest dev.local/agentstack-api:latest

agentstack-deploy: ## Deploy AgentStack full stack to K8s
	@echo "Deploying AgentStack to Kubernetes..."
	kubectl apply -f agentstack-k8s.yaml

agentstack-status: ## Check AgentStack status
	@echo "📊 AgentStack Status"
	@echo "───────────────────"
	@kubectl get pods -n agentstack
	@echo ""
	@echo "🌐 Services"
	@echo "──────────"
	@kubectl get svc -n agentstack

agentstack-logs: ## View AgentStack API logs
	@kubectl logs -f deployment/agentstack-api -n agentstack

agentstack-worker-logs: ## View AgentStack Worker logs
	@kubectl logs -f deployment/agentstack-worker -n agentstack

##@ Installation

check-cluster: ## Check Kubernetes connection
	@printf "Checking cluster connection... "
	@kubectl cluster-info >/dev/null 2>&1 && printf '\033[0;32m✓\033[0m\n' || { printf '\033[0;31m✗\033[0m\n'; echo "Please start Kubernetes (OrbStack/Docker Desktop/minikube)"; exit 1; }

check-api-key: ## Check OpenAI API key
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo ""; \
		printf '\033[1;33m⚠ OPENAI_API_KEY not set\033[0m\n'; \
		echo ""; \
		echo "Set your OpenAI API key:"; \
		echo "  export OPENAI_API_KEY='your-key-here'"; \
		echo ""; \
		echo "Or continue without (you can configure later in the UI)"; \
		read -p "Continue anyway? [y/N]: " CONTINUE; \
		if [ "$$CONTINUE" != "y" ] && [ "$$CONTINUE" != "Y" ]; then exit 1; fi; \
	else \
		printf "OpenAI API key... \033[0;32m✓\033[0m\n"; \
	fi

install-cli: ## Install kagent CLI
	@printf "Installing kagent CLI... "
	@if command -v kagent >/dev/null 2>&1; then \
		printf '\033[0;32m✓ already installed\033[0m\n'; \
	else \
		if command -v brew >/dev/null 2>&1; then \
			brew install kagent >/dev/null 2>&1 && printf '\033[0;32m✓\033[0m\n' || { \
				curl -sL https://raw.githubusercontent.com/kagent-dev/kagent/refs/heads/main/scripts/get-kagent | bash >/dev/null 2>&1 && printf '\033[0;32m✓\033[0m\n'; \
			}; \
		else \
			curl -sL https://raw.githubusercontent.com/kagent-dev/kagent/refs/heads/main/scripts/get-kagent | bash >/dev/null 2>&1 && printf '\033[0;32m✓\033[0m\n'; \
		fi; \
	fi

install: install-cli ## Alias for install-cli

setup-kagent: ## Install kagent to cluster
	@echo "Installing kagent ($(KAGENT_PROFILE) profile)..."
	@if helm list -n $(AGENT_NAMESPACE) 2>/dev/null | grep -q kagent; then \
		printf '\033[0;32m✓ Kagent already installed\033[0m\n'; \
	else \
		if [ -n "$$OPENAI_API_KEY" ]; then \
			kagent install --profile $(KAGENT_PROFILE) -n $(AGENT_NAMESPACE); \
		else \
			kagent install --profile $(KAGENT_PROFILE) -n $(AGENT_NAMESPACE) 2>/dev/null || \
			( \
				helm install kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds -n $(AGENT_NAMESPACE) --create-namespace && \
				helm install kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent -n $(AGENT_NAMESPACE) --set ui.enabled=true \
			); \
		fi; \
		printf '\033[0;32m✓ Kagent installed\033[0m\n'; \
	fi

configure-api-key: ## Configure OpenAI API key for agents
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo ""; \
		printf '\033[0;31m✗ OPENAI_API_KEY not set\033[0m\n'; \
		echo "Please run: export OPENAI_API_KEY='your-key-here'"; \
		exit 1; \
	fi
	@printf "Configuring OpenAI API key... "
	@kubectl create secret generic kagent-openai \
		-n $(AGENT_NAMESPACE) \
		--from-literal=OPENAI_API_KEY="$$OPENAI_API_KEY" \
		--dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
	@printf '\033[0;32m✓\033[0m\n'
	@echo "Restarting agents to apply configuration..."
	@kubectl rollout restart deployment -n $(AGENT_NAMESPACE) -l app.kubernetes.io/part-of=kagent >/dev/null 2>&1 || true
	@printf '\033[0;32m✓ Agents will restart automatically\033[0m\n'

setup: check-cluster setup-kagent configure-api-key ## Install kagent (requires cluster)

##@ Management

agents: ## List all agents
	@kubectl get agents -n $(AGENT_NAMESPACE) 2>/dev/null || echo "No agents found"

logs: ## View kagent logs
	@kubectl logs -n $(AGENT_NAMESPACE) -l app.kubernetes.io/component=controller --tail=50 -f

invoke: ## Chat with an agent (usage: make invoke AGENT=k8s-agent)
	@if [ -z "$(AGENT)" ]; then \
		echo "Usage: make invoke AGENT=<agent-name>"; \
		echo ""; \
		echo "Available agents:"; \
		kagent get agent 2>/dev/null || kubectl get agents -n $(AGENT_NAMESPACE) -o custom-columns=NAME:.metadata.name --no-headers; \
	else \
		kagent invoke --agent $(AGENT); \
	fi

##@ Cleanup

clean: ## Remove kagent
	@echo "Removing kagent..."
	@kagent uninstall -n $(AGENT_NAMESPACE) 2>/dev/null || { \
		helm uninstall kagent -n $(AGENT_NAMESPACE) 2>/dev/null || true; \
		helm uninstall kagent-crds -n $(AGENT_NAMESPACE) 2>/dev/null || true; \
	}
	@printf '\033[0;32m✓ Kagent removed\033[0m\n'

uninstall: clean ## Alias for clean

clean-all: ## Remove everything including namespace
	@echo "Removing all kagent resources..."
	@kagent uninstall -n $(AGENT_NAMESPACE) 2>/dev/null || true
	@helm uninstall kagent -n $(AGENT_NAMESPACE) 2>/dev/null || true
	@helm uninstall kagent-crds -n $(AGENT_NAMESPACE) 2>/dev/null || true
	@kubectl delete namespace $(AGENT_NAMESPACE) 2>/dev/null || true
	@printf '\033[0;32m✓ Complete cleanup done\033[0m\n'

##@ Advanced

adk-agent: ## Build and deploy Google ADK agent
	@echo "Building ADK agent Docker image..."
	@cd kagent-adk-agent && docker build -t dev.local/kagent-adk-agent:latest .
	@printf '\033[0;32m✓ Image built\033[0m\n'
	@echo "Deploying ADK agent to kagent..."
	@kubectl apply -f kagent-adk-agent/kagent-deployment.yaml
	@printf '\033[0;32m✓ ADK agent deployed\033[0m\n'
	@echo "Waiting for agent to be ready (this may take 1-2 minutes)..."
	@sleep 5
	@kubectl rollout status deployment/google-adk-byo-agent -n $(AGENT_NAMESPACE) --timeout=120s 2>/dev/null || echo "⚠️  Agent pod starting... Check with: make adk-agent-status"

adk-agent-build: ## Build ADK agent Docker image
	@echo "Building ADK agent Docker image..."
	@cd kagent-adk-agent && docker build -t dev.local/kagent-adk-agent:latest .
	@printf '\033[0;32m✓ Image built: dev.local/kagent-adk-agent:latest\033[0m\n'

adk-agent-deploy: ## Deploy ADK agent to kagent
	@echo "Deploying ADK agent to kagent..."
	@kubectl apply -f kagent-adk-agent/kagent-deployment.yaml
	@printf '\033[0;32m✓ ADK agent deployed\033[0m\n'
	@echo "Check status with: make adk-agent-status"

adk-agent-status: ## Check ADK agent status
	@echo "📦 ADK Agent Status"
	@echo "───────────────────"
	@kubectl get agent google-adk-byo-agent -n $(AGENT_NAMESPACE) 2>/dev/null || echo "Agent not found"
	@echo ""
	@echo "🔧 Deployment"
	@echo "─────────────"
	@kubectl get deployment google-adk-byo-agent -n $(AGENT_NAMESPACE) 2>/dev/null || echo "Deployment not found"
	@echo ""
	@echo "📊 Pods"
	@echo "───────"
	@kubectl get pods -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=google-adk-byo-agent 2>/dev/null || echo "No pods found"

adk-agent-logs: ## View ADK agent logs
	@kubectl logs -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=google-adk-byo-agent --tail=100 -f

adk-agent-delete: ## Delete ADK agent
	@echo "Removing ADK agent..."
	@kubectl delete -f kagent-adk-agent/kagent-deployment.yaml 2>/dev/null || echo "Agent not found"
	@printf '\033[0;32m✓ ADK agent removed\033[0m\n'

knative: ## Install Knative (optional, for serverless agents)
	@echo "Installing Knative Serving..."
	@./knative_orbstack.sh
	@printf '\033[0;32m✓ Knative installed\033[0m\n'

debug: ## Show debug information
	@echo "=== Cluster Info ==="
	@kubectl cluster-info
	@echo ""
	@echo "=== Kagent Pods ==="
	@kubectl get pods -n $(AGENT_NAMESPACE) -o wide 2>/dev/null || echo "No pods"
	@echo ""
	@echo "=== Kagent Services ==="
	@kubectl get svc -n $(AGENT_NAMESPACE) 2>/dev/null || echo "No services"
	@echo ""
	@echo "=== Events ==="
	@kubectl get events -n $(AGENT_NAMESPACE) --sort-by='.lastTimestamp' | tail -10 2>/dev/null || echo "No events"

.PHONY: all
all: help
