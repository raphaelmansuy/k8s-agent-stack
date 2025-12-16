# Prompt: k8s-agent-stack Operational Readiness & Documentation Overhaul

## Objective

Ensure the **k8s-agent-stack** is fully operational, production-ready, and easy for users to deploy agents on.  
Your primary deliverable is a **concise, accurate, and high-value documentation set** that reflects the **current state of the system** and enables users to get started quickly without wading through outdated or bloated material.

---

## Scope of Work

### 1. Stack Verification & Readiness

- Verify that the **k8s-agent-stack is fully deployed and operational**
- Validate all deployment scripts and manifests
- Confirm all required components:
  - Start successfully
  - Are healthy and observable
  - Are accessible via documented endpoints or interfaces
- Identify and resolve any issues preventing agent deployment

---

### 2. Documentation Audit & Cleanup

- Audit **all existing documentation**
  - Identify outdated, redundant, or inaccurate content
- Archive all existing documentation to:
  ```
  archive/old-docs/<timestamp>/
  ```
  - Preserve history while ensuring a clean slate

---

### 3. New Documentation Set (Required Deliverables)

Produce a **concise, high-signal documentation set** designed for fast onboarding and operational clarity.

#### Required Sections

1. **Getting Started**
   - Prerequisites
   - Supported environments
   - Minimal setup steps

2. **Architecture Overview**
   - High-value **ASCII diagrams** explaining:
     - Core components
     - Data/control flow
     - Agent lifecycle
   - Clear explanation of component responsibilities and relationships

3. **Deployment Guide**
   - Step-by-step agent deployment instructions
   - Clear access methods (CLI, API, UI, etc.)
   - Environment-specific notes if applicable

4. **Verification & Health Checks**
   - Commands and steps to confirm:
     - Stack health
     - Agent readiness
     - Successful deployments

5. **Troubleshooting**
   - Common failure modes
   - Symptoms → causes → fixes
   - Logs, commands, and diagnostics users should check

6. **Quick Reference**
   - Common commands
   - Operational workflows
   - Day-2 operations (restart, scale, upgrade, remove agents)

7. **Known Limitations & Workarounds**
   - Current constraints
   - Temporary mitigations
   - Clear notes on what is *not* supported

8. **Glossary & Further Reading**
   - Definitions of key terms and concepts
   - Links to:
     - Official Kubernetes docs
     - Related component documentation
     - Relevant tutorials or specifications

---

## Accuracy & Quality Bar

- Documentation **must reflect the actual current behavior** of the system
- Instructions must be:
  - Explicit
  - Reproducible
  - Easy to follow by a new user
- Prefer clarity and correctness over completeness
- Avoid speculative or aspirational content

---

## Working Methodology

### ODA Loop (Observe → Decide → Act)

Use an iterative approach:
1. **Observe** the current system, scripts, and documentation
2. **Decide** what must change or be improved
3. **Act** by fixing, validating, or documenting

Repeat as needed until the stack and docs are aligned.

---

## Process Artifacts (Required)

### 1. Scratchpad

- File: `./process/scratchpad.md`
- Purpose:
  - Notes
  - Observations
  - Hypotheses
  - Decision rationale
- Markdown format

### 2. Plan & Execution Log

- File: `./process/plan.md`
- Purpose:
  - Track:
    - What has been done
    - What remains
    - Issues encountered
    - Fixes applied
    - Technical insights
- Acts as a living state and audit log

---

## Success Criteria

- ✅ Stack passes all verification checks
- ✅ Agents can be deployed reliably using documented steps
- ✅ Documentation is concise, accurate, and actionable
- ✅ New users can deploy an agent without external guidance
- ✅ Known limitations are clearly documented
- ✅ All prior documentation is safely archived
