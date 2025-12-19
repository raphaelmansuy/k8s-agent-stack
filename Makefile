# k8s-agent-stack Makefile
# Copyright 2025 Raphaël MANSUY
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: help start install setup ui mlflow-ui status clean uninstall check-deps check-cluster check-api-key docs install-tools

# --- Variables ---
AGENT_NAMESPACE := kagent
KAGENT_PROFILE   := demo
SHELL           := /bin/bash

# --- Colors ---
BLUE         := \033[0;34m
GREEN        := \033[0;32m
RED          := \033[0;31m
YELLOW       := \033[1;33m
CYAN         := \033[0;36m
BOLD         := \033[1m
NC           := \033[0m # No Color

# --- UI Helpers ---
define print_header
	printf "$(BLUE)╔════════════════════════════════════════════════════════════════╗$(NC)\n"
	printf "$(BLUE)║$(BOLD)         k8s-agent-stack - AI Agent Platform                    $(NC)$(BLUE)║$(NC)\n"
	printf "$(BLUE)╚════════════════════════════════════════════════════════════════╝$(NC)\n"
endef

define print_success
	printf "$(GREEN)✓ %s$(NC)\n" "$(1)"
endef

define print_error
	printf "$(RED)✗ %s$(NC)\n" "$(1)"
endef

define print_warning
	printf "$(YELLOW)⚠ %s$(NC)\n" "$(1)"
endef

define print_step
	printf "$(CYAN)➜ %s...$(NC) " "$(1)"
endef

# --- Default Target ---
.DEFAULT_GOAL := help

##@ 🚀 Quick Start

help: ## ℹ️  Show this help message
	$(print_header)
	@echo ""
	@echo -e "$(BOLD)🚀 Quick Start$(NC)"
	@echo -e "  $(CYAN)make start$(NC)         Complete setup (one command!)"
	@echo -e "  $(CYAN)make ui$(NC)            Open Kagent web interface"
	@echo -e "  $(CYAN)make mlflow-ui$(NC)     Open MLflow interface"
	@echo -e "  $(CYAN)make docs$(NC)          Open API documentation"
	@echo -e "  $(CYAN)make status$(NC)        Check everything is running"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "$(BOLD)📋 Command Groups$(NC)\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-18s$(NC) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n$(BOLD)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo -e "$(BOLD)💡 Tip:$(NC) Run '$(CYAN)make check-deps$(NC)' to verify your environment."
	@echo ""

start: check-deps check-cluster check-api-key install-cli setup-kagent configure-api-key ## ⚡ Complete setup in one command
	@echo ""
	$(call print_success,Setup complete!)
	@echo ""
	@echo -e "$(BOLD)Next steps:$(NC)"
	@echo -e "  $(CYAN)make ui$(NC)        → Open Kagent web interface"
	@echo -e "  $(CYAN)make mlflow-ui$(NC) → Open MLflow interface"
	@echo -e "  $(CYAN)make docs$(NC)      → Open API documentation"
	@echo -e "  $(CYAN)make status$(NC)    → Check component status"
	@echo -e "  $(CYAN)make agents$(NC)    → List all agents"
	@echo ""

status: ## 📊 Check cluster and kagent status
	@echo -e "$(BOLD)📊 Cluster Status$(NC)"
	@echo "────────────────"
	@kubectl cluster-info 2>/dev/null | head -1 || $(call print_error,Not connected to cluster)
	@echo ""
	@echo -e "$(BOLD)📦 Kagent Components$(NC)"
	@echo "────────────────────"
	@kubectl get pods -n $(AGENT_NAMESPACE) 2>/dev/null || $(call print_error,Kagent not installed)
	@echo ""
	@echo -e "$(BOLD)🤖 Agents$(NC)"
	@echo "─────────"
	@kubectl get agents -n $(AGENT_NAMESPACE) 2>/dev/null || echo "  No agents found"
	@echo ""

##@ 🌐 User Interfaces

ui: ## 🖥️  Open Kagent UI (http://localhost:8080)
	@printf "$(GREEN)╔════════════════════════════════════════╗$(NC)\n"
	@printf "$(GREEN)║  Kagent UI → http://localhost:8080    ║$(NC)\n"
	@printf "$(GREEN)╚════════════════════════════════════════╝$(NC)\n"
	@echo ""
	@echo -e "$(CYAN)🌐 Starting port-forward...$(NC)"
	@echo "   Keep this terminal open while using the UI"
	@echo "   Press Ctrl+C to stop"
	@echo ""
	@kubectl port-forward -n $(AGENT_NAMESPACE) svc/kagent-ui 8080:8080

mlflow-ui: ## 📈 Open MLflow UI (http://localhost:5000)
	@printf "$(GREEN)╔════════════════════════════════════════╗$(NC)\n"
	@printf "$(GREEN)║  MLflow UI → http://localhost:5000    ║$(NC)\n"
	@printf "$(GREEN)╚════════════════════════════════════════╝$(NC)\n"
	@echo ""
	@echo -e "$(CYAN)🌐 Starting port-forward for MLflow...$(NC)"
	@echo "   Keep this terminal open while using the UI"
	@echo "   Press Ctrl+C to stop"
	@echo ""
	@ (sleep 2 && open http://localhost:5000) &
	@kubectl port-forward -n agentstack svc/agentstack-mlflow 5000:5000

agentstack-ui: ## 🛠️  Open AgentStack Web UI (agentctl)
	$(call print_step,Opening AgentStack UI)
	@cd agentstack/cli && go run ./cmd/agentctl ui

docs: ## 📖 Open AgentStack API Documentation
	$(call print_step,Opening API Documentation)
	@echo -e "$(CYAN)🌐 Starting port-forward for AgentStack API...$(NC)"
	@echo "   API will be available at http://localhost:8082"
	@echo "   Press Ctrl+C to stop"
	@echo ""
	@ (sleep 2 && cd agentstack/cli && go run ./cmd/agentctl docs --endpoint http://localhost:8082) &
	@kubectl port-forward -n agentstack svc/agentstack-api 8082:8080

##@ 🤖 AgentStack API & SDK

agentstack-build: ## 🏗️  Build AgentStack API and Worker images
	$(call print_step,Building AgentStack images)
	@cd agentstack && make docker-build
	@docker tag agentstack-api:latest dev.local/agentstack-api:latest
	$(call print_success,Images built and tagged)

agentstack-deploy: ## 🚀 Deploy AgentStack full stack to K8s
	$(call print_step,Deploying AgentStack to Kubernetes)
	@kubectl apply -f deploy/agentstack-k8s.yaml
	$(call print_success,Deployment applied)

agentstack-status: ## 📈 Check AgentStack status
	@echo -e "$(BOLD)📊 AgentStack Status$(NC)"
	@echo "───────────────────"
	@kubectl get pods -n agentstack
	@echo ""
	@echo -e "$(BOLD)🌐 Services$(NC)"
	@echo "──────────"
	@kubectl get svc -n agentstack

agentstack-logs: ## 📝 View AgentStack API logs
	@kubectl logs -f deployment/agentstack-api -n agentstack

agentstack-worker-logs: ## 📝 View AgentStack Worker logs
	@kubectl logs -f deployment/agentstack-worker -n agentstack

agentstack-test: ## 🧪 Run AgentStack tests
	$(call print_step,Running AgentStack tests)
	@cd agentstack && make test

agentstack-lint: ## 🔍 Run AgentStack linters
	$(call print_step,Running AgentStack linters)
	@cd agentstack && make lint

##@ 🛠️  Installation & Setup

check-deps: ## 🔍 Check for required dependencies
	$(call print_step,Checking dependencies)
	@command -v kubectl >/dev/null 2>&1 || { $(call print_error,kubectl is required); exit 1; }
	@command -v docker >/dev/null 2>&1 || { $(call print_warning,docker is recommended); }
	@command -v helm >/dev/null 2>&1 || { $(call print_error,helm is required); exit 1; }
	@command -v go >/dev/null 2>&1 || { $(call print_error,go is required); exit 1; }
	$(call print_success,All dependencies found)

check-cluster: ## 🔍 Check Kubernetes connection
	$(call print_step,Checking cluster connection)
	@kubectl cluster-info >/dev/null 2>&1 && $(call print_success,Connected) || { echo ""; $(call print_error,Not connected to cluster. Please start Kubernetes (OrbStack/Docker Desktop/minikube)); exit 1; }

check-api-key: ## 🔑 Check OpenAI API key
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo ""; \
		$(call print_warning,OPENAI_API_KEY not set); \
		echo ""; \
		echo "Set your OpenAI API key:"; \
		echo "  export OPENAI_API_KEY='your-key-here'"; \
		echo ""; \
		echo "Or continue without (you can configure later in the UI)"; \
		read -p "Continue anyway? [y/N]: " CONTINUE; \
		if [ "$$CONTINUE" != "y" ] && [ "$$CONTINUE" != "Y" ]; then exit 1; fi; \
	else \
		$(call print_step,OpenAI API key); $(call print_success,Found); \
	fi

install-cli: ## 📥 Install kagent CLI
	$(call print_step,Installing kagent CLI)
	@if command -v kagent >/dev/null 2>&1; then \
		$(call print_success,Already installed); \
	else \
		if command -v brew >/dev/null 2>&1; then \
			brew install kagent >/dev/null 2>&1 && $(call print_success,Installed via brew) || { \
				curl -sL https://raw.githubusercontent.com/kagent-dev/kagent/refs/heads/main/scripts/get-kagent | bash >/dev/null 2>&1 && $(call print_success,Installed via script); \
			}; \
		else \
			curl -sL https://raw.githubusercontent.com/kagent-dev/kagent/refs/heads/main/scripts/get-kagent | bash >/dev/null 2>&1 && $(call print_success,Installed via script); \
		fi; \
	fi

setup-kagent: ## ⚙️  Install kagent to cluster
	$(call print_step,Installing kagent ($(KAGENT_PROFILE) profile))
	@if helm list -n $(AGENT_NAMESPACE) 2>/dev/null | grep -q kagent; then \
		$(call print_success,Kagent already installed); \
	else \
		if [ -n "$$OPENAI_API_KEY" ]; then \
			kagent install --profile $(KAGENT_PROFILE) -n $(AGENT_NAMESPACE) >/dev/null 2>&1; \
		else \
			kagent install --profile $(KAGENT_PROFILE) -n $(AGENT_NAMESPACE) 2>/dev/null || \
			( \
				helm install kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds -n $(AGENT_NAMESPACE) --create-namespace >/dev/null 2>&1 && \
				helm install kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent -n $(AGENT_NAMESPACE) --set ui.enabled=true >/dev/null 2>&1 \
			); \
		fi; \
		$(call print_success,Kagent installed); \
	fi

configure-api-key: ## 🔑 Configure OpenAI API key for agents
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		$(call print_warning,OPENAI_API_KEY not set. Skipping secret creation.); \
	else \
		$(call print_step,Configuring OpenAI API key); \
		@kubectl create secret generic kagent-openai \
			-n $(AGENT_NAMESPACE) \
			--from-literal=OPENAI_API_KEY="$$OPENAI_API_KEY" \
			--dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1; \
		$(call print_success,Secret created); \
		$(call print_step,Restarting agents); \
		@kubectl rollout restart deployment -n $(AGENT_NAMESPACE) -l app.kubernetes.io/part-of=kagent >/dev/null 2>&1 || true; \
		$(call print_success,Restart triggered); \
	fi

##@ 📋 Management

agents: ## 🤖 List all agents
	@kubectl get agents -n $(AGENT_NAMESPACE) 2>/dev/null || echo "No agents found"

logs: ## 📝 View kagent controller logs
	@kubectl logs -n $(AGENT_NAMESPACE) -l app.kubernetes.io/component=controller --tail=50 -f

invoke: ## 💬 Chat with an agent (usage: make invoke AGENT=k8s-agent)
	@if [ -z "$(AGENT)" ]; then \
		echo "Usage: make invoke AGENT=<agent-name>"; \
		echo ""; \
		echo "Available agents:"; \
		kagent get agent 2>/dev/null || kubectl get agents -n $(AGENT_NAMESPACE) -o custom-columns=NAME:.metadata.name --no-headers; \
	else \
		kagent invoke --agent $(AGENT); \
	fi

##@ 🧹 Cleanup

clean: ## 🗑️  Remove kagent from cluster
	$(call print_step,Removing kagent)
	@kagent uninstall -n $(AGENT_NAMESPACE) 2>/dev/null || { \
		helm uninstall kagent -n $(AGENT_NAMESPACE) 2>/dev/null || true; \
		helm uninstall kagent-crds -n $(AGENT_NAMESPACE) 2>/dev/null || true; \
	}
	$(call print_success,Kagent removed)

clean-all: clean ## 🧨 Remove everything including namespace
	$(call print_step,Deleting namespace $(AGENT_NAMESPACE))
	@kubectl delete namespace $(AGENT_NAMESPACE) 2>/dev/null || true
	$(call print_success,Complete cleanup done)

##@ 🧪 Advanced & Experimental

adk-agent: adk-agent-build adk-agent-deploy ## 🧪 Build and deploy Google ADK agent

adk-agent-build: ## 🏗️  Build ADK agent Docker image
	$(call print_step,Building ADK agent image)
	@cd kagent-adk-agent && docker build -t dev.local/kagent-adk-agent:latest . >/dev/null 2>&1
	$(call print_success,Image built: dev.local/kagent-adk-agent:latest)

adk-agent-deploy: ## 🚀 Deploy ADK agent to kagent
	$(call print_step,Deploying ADK agent)
	@kubectl apply -f deploy/kagent-adk-agent.yaml >/dev/null 2>&1
	$(call print_success,ADK agent deployed)
	@echo "Check status with: make adk-agent-status"

adk-agent-status: ## 📈 Check ADK agent status
	@echo -e "$(BOLD)📦 ADK Agent Status$(NC)"
	@echo "───────────────────"
	@kubectl get agent google-adk-byo-agent -n $(AGENT_NAMESPACE) 2>/dev/null || echo "Agent not found"
	@echo ""
	@kubectl get pods -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=google-adk-byo-agent 2>/dev/null || echo "No pods found"

adk-agent-logs: ## 📝 View ADK agent logs
	@kubectl logs -n $(AGENT_NAMESPACE) -l app.kubernetes.io/name=google-adk-byo-agent --tail=100 -f

##@ 🛠️  Development

install-tools: ## 🛠️  Install development tools (Go, linters, etc.)
	$(call print_step,Installing development tools)
	@cd agentstack && $(MAKE) install-tools
	$(call print_success,Development tools installed)

knative: ## ⚡ Install Knative (optional)
	$(call print_step,Installing Knative Serving)
	@./scripts/knative_orbstack.sh
	$(call print_success,Knative installed)

debug: ## 🔍 Show debug information
	@echo -e "$(BOLD)=== Cluster Info ===$(NC)"
	@kubectl cluster-info
	@echo ""
	@echo -e "$(BOLD)=== Kagent Pods ===$(NC)"
	@kubectl get pods -n $(AGENT_NAMESPACE) -o wide 2>/dev/null || echo "No pods"
	@echo ""
	@echo -e "$(BOLD)=== Events ===$(NC)"
	@kubectl get events -n $(AGENT_NAMESPACE) --sort-by='.lastTimestamp' | tail -10 2>/dev/null || echo "No events"

.PHONY: all
all: help
