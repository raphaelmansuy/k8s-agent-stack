# k8s-agent-stack Makefile
# Copyright 2025 Raphaël MANSUY
# Licensed under the Apache License, Version 2.0

.PHONY: help install setup deploy test build logs clean verify dev prod port-forward list-agents agent-logs

# Variables
PROJECT_NAME := k8s-agent-stack
AGENT_NAME := google-adk-agent
AGENT_NAMESPACE := kagent
DOCKER_REGISTRY := gcr.io/your-project
AGENT_IMAGE := $(DOCKER_REGISTRY)/$(AGENT_NAME)
AGENT_VERSION := v1.0.0
PYTHON := python3

# Color output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help screen
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║         k8s-agent-stack - AI Agent Platform on Kubernetes      ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(BLUE)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)Examples:$(NC)"
	@echo "  make setup              # Install Knative + kagent locally"
	@echo "  make deploy             # Deploy example agent"
	@echo "  make test               # Run all tests"
	@echo "  make clean              # Remove agents and resources"
	@echo ""

##@ Installation & Setup

install: ## Install required tools (kn, kubectl, helm, docker)
	@echo "$(BLUE)Installing required tools...$(NC)"
	@command -v kubectl >/dev/null 2>&1 || { echo "$(RED)kubectl not found$(NC)"; exit 1; }
	@command -v kn >/dev/null 2>&1 || { echo "$(YELLOW)Installing kn CLI...$(NC)" && brew install knative/client/kn; }
	@command -v helm >/dev/null 2>&1 || { echo "$(YELLOW)Installing helm...$(NC)" && brew install helm; }
	@command -v docker >/dev/null 2>&1 || { echo "$(YELLOW)Installing docker...$(NC)" && brew install --cask docker; }
	@echo "$(GREEN)✓ All tools installed$(NC)"

setup: ## Setup local Kubernetes with Knative and kagent (OrbStack or kind)
	@echo "$(BLUE)Setting up local Kubernetes environment...$(NC)"
	@echo "$(YELLOW)Step 1: Verifying Kubernetes cluster...$(NC)"
	@kubectl cluster-info >/dev/null || { echo "$(RED)Kubernetes cluster not found$(NC)"; exit 1; }
	@echo "$(YELLOW)Step 2: Verifying/installing kn CLI...$(NC)"
	@command -v kn >/dev/null 2>&1 || { echo "$(YELLOW)Installing kn CLI...$(NC)"; brew install knative/client/kn 2>/dev/null || echo "$(YELLOW)kn CLI install skipped (may already exist)"; }
	@echo "$(YELLOW)Step 3: Running Knative + kagent installer...$(NC)"
	@./knative_orbstack.sh
	@echo "$(YELLOW)Step 4: Verifying installation...$(NC)"
	@$(MAKE) verify
	@echo "$(GREEN)✓ Setup complete! Ready to deploy agents.$(NC)"

setup-metrics: ## Setup local K8s with metrics server for autoscaling
	@echo "$(BLUE)Installing metrics server...$(NC)"
	@./knative_orbstack.sh --install-metrics
	@echo "$(GREEN)✓ Metrics server installed$(NC)"

setup-warm: ## Setup with warm pods (no scale-to-zero)
	@echo "$(BLUE)Installing with warm pods...$(NC)"
	@./knative_orbstack.sh --warm 1
	@echo "$(GREEN)✓ Setup with warm pods complete$(NC)"

verify: ## Verify installation status
	@echo "$(BLUE)Verifying installation...$(NC)"
	@echo ""
	@echo "$(YELLOW)Knative Serving:$(NC)"
	@kubectl get ns knative-serving >/dev/null 2>&1 && echo "  $(GREEN)✓ Namespace exists$(NC)" || echo "  $(RED)✗ Not installed$(NC)"
	@kubectl get pods -n knative-serving >/dev/null 2>&1 && echo "  $(GREEN)✓ Pods running$(NC)" || echo "  $(RED)✗ Pods not found$(NC)"
	@echo ""
	@echo "$(YELLOW)Contour/Envoy:$(NC)"
	@kubectl get ns projectcontour >/dev/null 2>&1 && echo "  $(GREEN)✓ Namespace exists$(NC)" || echo "  $(RED)✗ Not installed$(NC)"
	@kubectl get svc envoy -n projectcontour -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null | grep -q . && echo "  $(GREEN)✓ Envoy IP assigned$(NC)" || echo "  $(YELLOW)⚠ Envoy IP pending (may take a moment)$(NC)"
	@echo ""
	@echo "$(YELLOW)kagent:$(NC)"
	@kubectl get ns kagent >/dev/null 2>&1 && echo "  $(GREEN)✓ Namespace exists$(NC)" || echo "  $(RED)✗ Not installed$(NC)"
	@kubectl get crds 2>/dev/null | grep -q "kagent.dev" && echo "  $(GREEN)✓ CRDs installed$(NC)" || echo "  $(RED)✗ CRDs not found$(NC)"
	@echo ""
	@echo "$(YELLOW)Knative Services:$(NC)"
	@kubectl get ksvc -n default 2>/dev/null | tail -n +2 | wc -l | awk '{if ($$1 > 0) print "  $(GREEN)✓ "$$1" service(s) ready$(NC)"; else print "  $(YELLOW)⚠ No sample services deployed yet$(NC)"}'
	@echo ""

##@ Agent Deployment

deploy: ## Deploy example Google ADK agent
	@echo "$(BLUE)Deploying Google ADK agent...$(NC)"
	@kubectl apply -f kagent-setup.yaml
	@echo "$(YELLOW)Waiting for agent to be ready...$(NC)"
	@kubectl wait --for=condition=Ready ksvc/google-adk-agent \
		-n kagent --timeout=120s 2>/dev/null || echo "$(YELLOW)Agent initializing...$(NC)"
	@echo "$(GREEN)✓ Agent deployed$(NC)"
	@$(MAKE) agent-status

deploy-k8s-helper: ## Deploy Kubernetes helper agent
	@echo "$(BLUE)Deploying K8s helper agent...$(NC)"
	@kubectl apply -f examples/k8s-helper-agent.yaml
	@echo "$(GREEN)✓ K8s helper agent deployed$(NC)"

update-agent: ## Update agent image version
	@echo "$(YELLOW)Current agents:$(NC)"
	@kn service list
	@echo ""
	@read -p "Enter service name: " SERVICE_NAME; \
	read -p "Enter new image: " NEW_IMAGE; \
	kn service update $$SERVICE_NAME --image $$NEW_IMAGE

undeploy: ## Remove example agent
	@echo "$(RED)Removing Google ADK agent...$(NC)"
	@kubectl delete -f kagent-adk-agent/kagent-deployment.yaml 2>/dev/null || true
	@echo "$(GREEN)✓ Agent removed$(NC)"

##@ Testing & Building

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	@cd kagent-adk-agent && $(PYTHON) -m pytest tests/ -v

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	@cd kagent-adk-agent && $(PYTHON) -m pytest tests/unit/ -v

test-integration: ## Run integration tests only
	@echo "$(BLUE)Running integration tests...$(NC)"
	@cd kagent-adk-agent && $(PYTHON) -m pytest tests/integration/ -v

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@cd kagent-adk-agent && $(PYTHON) -m pytest tests/ -v --cov=app --cov-report=html
	@echo "$(GREEN)✓ Coverage report: $(PWD)/htmlcov/index.html$(NC)"

build: ## Build agent Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	@cd kagent-adk-agent && docker build -t $(AGENT_IMAGE):$(AGENT_VERSION) .
	@echo "$(GREEN)✓ Image built: $(AGENT_IMAGE):$(AGENT_VERSION)$(NC)"

build-dev: ## Build agent image for local development
	@echo "$(BLUE)Building development Docker image...$(NC)"
	@cd kagent-adk-agent && docker build -t $(AGENT_NAME):dev .
	@echo "$(GREEN)✓ Dev image built: $(AGENT_NAME):dev$(NC)"

push: ## Push Docker image to registry
	@echo "$(BLUE)Pushing image to registry...$(NC)"
	@docker push $(AGENT_IMAGE):$(AGENT_VERSION)
	@echo "$(GREEN)✓ Image pushed: $(AGENT_IMAGE):$(AGENT_VERSION)$(NC)"

##@ Monitoring & Debugging

list-agents: ## List all deployed agents
	@echo "$(BLUE)Deployed agents:$(NC)"
	@kn service list

agent-status: ## Check agent deployment status
	@echo "$(BLUE)Agent status:$(NC)"
	@kubectl get ksvc google-adk-agent -n kagent 2>/dev/null || kubectl get pods -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=$(AGENT_NAME)

agent-logs: ## View agent logs (real-time)
	@echo "$(BLUE)Agent logs (Ctrl+C to exit):$(NC)"
	@POD=$$(kubectl get pods -n kagent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null); \
	if [ -z "$$POD" ]; then echo "$(RED)No agent pods found$(NC)"; exit 1; fi; \
	kubectl logs -n kagent $$POD -c user-container --tail=50 --follow

agent-logs-previous: ## View previous agent pod logs (after crash)
	@echo "$(BLUE)Previous agent logs:$(NC)"
	@POD=$$(kubectl get pods -n kagent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null); \
	if [ -z "$$POD" ]; then echo "$(RED)No agent pods found$(NC)"; exit 1; fi; \
	kubectl logs -n kagent $$POD -c user-container --previous

agent-describe: ## Describe agent pod details
	@echo "$(BLUE)Agent pod details:$(NC)"
	@POD=$$(kubectl get pods -n kagent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null); \
	if [ -z "$$POD" ]; then echo "$(RED)No agent pods found$(NC)"; exit 1; fi; \
	kubectl describe pod -n kagent $$POD | head -50

port-forward: ## Port-forward agent service for local testing (8080:8080)
	@echo "$(BLUE)Port-forwarding agent service...$(NC)"
	@echo "$(YELLOW)Access agent at: http://localhost:8080$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(NC)"
	@kubectl port-forward -n kagent svc/google-adk-agent-00001-private 8080:80

port-forward-custom: ## Port-forward with custom local port
	@read -p "Enter local port (default 8080): " PORT; \
	PORT=$${PORT:-8080}; \
	kubectl port-forward -n kagent svc/google-adk-agent-00001-private $$PORT:80

test-agent: ## Test agent endpoint (requires port-forward in another terminal)
	@echo "$(BLUE)Testing agent endpoint...$(NC)"
	@curl -X GET http://localhost:8080 -s | head -20
	@echo ""

watch-pods: ## Watch pod scaling in real-time
	@echo "$(BLUE)Watching pods (Ctrl+C to exit)...$(NC)"
	@watch -n 1 'kubectl get pods -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=$(AGENT_NAME) -o wide'

knative-logs: ## View Knative autoscaler logs
	@echo "$(BLUE)Knative autoscaler logs:$(NC)"
	@kubectl logs -n knative-serving deploy/autoscaler --tail=30 --follow

##@ Cleanup & Maintenance

clean: ## Remove all agents and reset cluster
	@echo "$(RED)Removing all deployed agents...$(NC)"
	@kubectl delete all -n $(AGENT_NAMESPACE) --all 2>/dev/null || true
	@echo "$(GREEN)✓ Agents removed$(NC)"

clean-all: ## Full cleanup - remove Knative, kagent, and everything
	@echo "$(RED)WARNING: This will remove Knative, kagent, and all agents!$(NC)"
	@read -p "Are you sure? (yes/no): " CONFIRM; \
	if [ "$$CONFIRM" = "yes" ]; then \
		echo "$(RED)Cleaning up...$(NC)"; \
		kubectl delete namespace knative-serving projectcontour kagent 2>/dev/null || true; \
		echo "$(GREEN)✓ Cleanup complete$(NC)"; \
	else \
		echo "$(YELLOW)Cleanup cancelled$(NC)"; \
	fi

debug: ## Run diagnostics and show debug info
	@echo "$(BLUE)Running diagnostics...$(NC)"
	@./knative_orbstack.sh --debug

##@ Development

dev: ## Start local development environment
	@echo "$(BLUE)Starting development environment...$(NC)"
	@echo "$(YELLOW)Step 1: Setting up Kubernetes...$(NC)"
	@$(MAKE) setup
	@echo ""
	@echo "$(YELLOW)Step 2: Building agent image...$(NC)"
	@$(MAKE) build-dev
	@echo ""
	@echo "$(YELLOW)Step 3: Deploying agent...$(NC)"
	@$(MAKE) deploy
	@echo ""
	@echo "$(YELLOW)Step 4: Port-forwarding...$(NC)"
	@echo "$(GREEN)✓ Development environment ready!$(NC)"
	@echo "$(YELLOW)Run 'make port-forward' in another terminal to access the agent$(NC)"

dev-watch: ## Watch for code changes and rebuild (requires watchmedo)
	@echo "$(BLUE)Watching for code changes...$(NC)"
	@echo "$(YELLOW)Install watchmedo: pip install watchdog[watchmedo]$(NC)"
	@cd kagent-adk-agent && watchmedo shell-command \
		--patterns="*.py" \
		--recursive \
		--command='make build-dev && make deploy' \
		app/

prod: ## Setup production environment
	@echo "$(BLUE)Setting up production environment...$(NC)"
	@$(MAKE) setup-metrics
	@echo "$(YELLOW)Configure your domain in Knative:$(NC)"
	@echo "  kubectl edit configmap config-domain -n knative-serving"
	@echo "$(YELLOW)Install cert-manager for TLS:$(NC)"
	@echo "  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml"
	@echo "$(GREEN)✓ Production setup ready$(NC)"

##@ Kagent Portal UI

portal-deploy: ## Deploy Kagent Portal UI for agent management
	@echo "$(BLUE)Deploying Kagent Portal...$(NC)"
	@kubectl apply -f kagent-portal.yaml
	@echo "$(GREEN)✓ Portal deployment manifest applied$(NC)"
	@echo "$(YELLOW)Next: Run 'make portal-access' to connect$(NC)"

portal-access: ## Access Kagent Portal (port-forward)
	@echo "$(BLUE)Starting Kagent Portal port-forward...$(NC)"
	@echo "$(GREEN)Portal will be available at: http://localhost:3000$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(NC)"
	@echo ""
	@kubectl port-forward -n $(AGENT_NAMESPACE) svc/kagent-web 3000:3000

portal-status: ## Check Kagent Portal status
	@echo "$(BLUE)Kagent Portal Status:$(NC)"
	@echo "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BLUE)Pods:$(NC)"
	@kubectl get pods -n $(AGENT_NAMESPACE) -l app=kagent-web
	@echo ""
	@echo "$(BLUE)Services:$(NC)"
	@kubectl get svc -n $(AGENT_NAMESPACE) -l app=kagent-web
	@echo ""
	@echo "$(BLUE)Access:$(NC)"
	@echo "  $(GREEN)Local: http://localhost:3000 (requires port-forward)$(NC)"
	@echo "  $(YELLOW)Run 'make portal-access' to start$(NC)"

portal-logs: ## Show Kagent Portal logs
	@echo "$(BLUE)Kagent Portal Logs:$(NC)"
	@kubectl logs -n $(AGENT_NAMESPACE) -l app=kagent-web -f

portal-clean: ## Remove Kagent Portal
	@echo "$(BLUE)Removing Kagent Portal...$(NC)"
	@kubectl delete -f kagent-portal.yaml --ignore-not-found
	@echo "$(GREEN)✓ Portal removed$(NC)"

##@ Documentation & Help

docs: ## Open documentation
	@echo "$(BLUE)Available documentation:$(NC)"
	@ls -lh *.md docs/*.md 2>/dev/null | awk '{print "  " $$9}'
	@echo ""
	@echo "$(YELLOW)Quick links:$(NC)"
	@echo "  - README.md - Main documentation"
	@echo "  - KAGENT_PORTAL_ACCESS.md - Portal access guide"
	@echo "  - knative-orbstack.md - Local development guide"
	@echo "  - knative.md - Production deployment guide"
	@echo "  - docs/kagent-adk-a2a-architecture.md - Architecture details"

version: ## Show version information
	@echo "$(BLUE)k8s-agent-stack $(AGENT_VERSION)$(NC)"
	@echo ""
	@echo "$(YELLOW)Component versions:$(NC)"
	@echo "  Knative: $$(kn version 2>/dev/null | head -1 || echo 'Not installed')"
	@echo "  Kubernetes: $$(kubectl version --short 2>/dev/null | grep Server || echo 'Not installed')"
	@echo "  Docker: $$(docker --version 2>/dev/null || echo 'Not installed')"
	@echo "  Helm: $$(helm version --short 2>/dev/null || echo 'Not installed')"

##@ Advanced

traffic-split: ## Example: Split traffic between agent versions (canary)
	@echo "$(BLUE)Canary deployment example:$(NC)"
	@echo ""
	@echo "  # Deploy new version:"
	@echo "  $$ kn service update $(AGENT_NAME) --image $(AGENT_IMAGE):v2"
	@echo ""
	@echo "  # Split traffic 90/10 (canary):"
	@echo "  $$ kn service update $(AGENT_NAME) \\"
	@echo "    --traffic $(AGENT_NAME)-v1=90,@latest=10"
	@echo ""
	@echo "  # Monitor metrics and gradually increase traffic:"
	@echo "  $$ kn service update $(AGENT_NAME) \\"
	@echo "    --traffic $(AGENT_NAME)-v1=50,@latest=50"
	@echo ""
	@echo "  # Full rollout:"
	@echo "  $$ kn service update $(AGENT_NAME) \\"
	@echo "    --traffic @latest=100"

scale-config: ## Show current scaling configuration
	@echo "$(BLUE)Current scaling configuration:$(NC)"
	@kubectl get ksvc -n $(AGENT_NAMESPACE) -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{.metadata.annotations.autoscaling\.knative\.dev/scale-min}{"\n"}{.metadata.annotations.autoscaling\.knative\.dev/scale-max}{"\n\n"}{end}'

##@ Utilities

shell: ## Open bash shell in agent pod
	@echo "$(BLUE)Opening shell in agent pod...$(NC)"
	@kubectl exec -it -n $(AGENT_NAMESPACE) \
		$$(kubectl get pod -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=$(AGENT_NAME) -o jsonpath='{.items[0].metadata.name}') -- /bin/bash

env: ## Show environment variables
	@echo "$(BLUE)Environment configuration:$(NC)"
	@echo "  PROJECT_NAME: $(PROJECT_NAME)"
	@echo "  AGENT_NAME: $(AGENT_NAME)"
	@echo "  AGENT_NAMESPACE: $(AGENT_NAMESPACE)"
	@echo "  DOCKER_REGISTRY: $(DOCKER_REGISTRY)"
	@echo "  AGENT_IMAGE: $(AGENT_IMAGE)"
	@echo "  AGENT_VERSION: $(AGENT_VERSION)"

.PHONY: all
all: help

