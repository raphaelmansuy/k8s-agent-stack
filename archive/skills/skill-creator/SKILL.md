---
name: skill-creator
description: Guide for creating effective Claude Skills for the AgentStack Sovereign Platform. Use this skill when creating new skills, updating existing skills, or learning skill design patterns. Triggers on requests like "create a new skill", "build a skill for X", "skill design", "skill template", or "extend Claude's capabilities".
---

# Skill Creator

## Overview

This skill provides comprehensive guidance for creating effective Claude Skills that extend capabilities with specialized knowledge, workflows, and tools for the AgentStack Sovereign Platform.

## Skill Anatomy

Every skill consists of a required SKILL.md file and optional bundled resources:

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter (name + description)
│   └── Markdown instructions
└── Bundled Resources (optional)
    ├── scripts/       - Executable code (Python/Bash)
    ├── references/    - Documentation loaded into context
    └── assets/        - Templates, icons, fonts for output
```

## Core Principles

### 1. Context Window is Precious

Claude's context window is shared. Only add information Claude doesn't already have.

- **Challenge each piece**: "Does Claude really need this?"
- **Prefer examples over explanations**: Show, don't tell
- **Keep SKILL.md under 500 lines**: Split into references if larger

### 2. Progressive Disclosure

Three-level loading system:

1. **Metadata** (always loaded): ~100 words - name + description
2. **SKILL.md body** (on trigger): <5k words - core workflow
3. **Bundled resources** (as needed): Unlimited - detailed docs

### 3. Degrees of Freedom

Match specificity to task fragility:

| Freedom Level | Use When | Example |
|---------------|----------|---------|
| **High** (text) | Multiple valid approaches | Brainstorming, design |
| **Medium** (pseudocode) | Preferred pattern exists | API integration |
| **Low** (scripts) | Fragile, must be exact | Database migrations |

## Skill Creation Workflow

### Step 1: Understand with Concrete Examples

Before creating, clarify:

- What specific queries should trigger this skill?
- What are 3-5 concrete usage examples?
- What should NOT trigger this skill?

Example questions:
- "What would a user say that should trigger this skill?"
- "Can you give examples of how this skill would be used?"

### Step 2: Plan Reusable Contents

Analyze each example to identify:

| Resource Type | Include When |
|---------------|--------------|
| `scripts/` | Same code written repeatedly, deterministic reliability needed |
| `references/` | Documentation Claude should reference while working |
| `assets/` | Templates, boilerplate to copy into output |

### Step 3: Initialize Skill

Run the initialization script:

```bash
python scripts/init_skill.py <skill-name> --path skills/
```

This creates:
- Skill directory with SKILL.md template
- Example scripts/, references/, assets/ directories
- Placeholder files to customize or delete

### Step 4: Write SKILL.md

#### Frontmatter (YAML)

```yaml
---
name: my-skill
description: Clear explanation of what the skill does AND when to use it. 
  Include triggers: specific scenarios, file types, or tasks. This is the 
  PRIMARY mechanism for skill activation - make it comprehensive.
---
```

**Critical**: All "when to use" information goes in description, NOT body.

#### Body (Markdown)

Use imperative form. Structure options:

**Workflow-Based** (sequential processes):
```markdown
## Workflow Decision Tree
## Step 1: [First Step]
## Step 2: [Second Step]
```

**Task-Based** (tool collections):
```markdown
## Quick Start
## Task Category 1
## Task Category 2
```

**Reference/Guidelines** (standards):
```markdown
## Guidelines
## Specifications
## Usage
```

### Step 5: Test Skill

Test activation:
- "List available skills" - verify it appears
- Try trigger phrases from description
- Verify correct resources load

### Step 6: Iterate

1. Use skill on real tasks
2. Note struggles or inefficiencies
3. Update SKILL.md or resources
4. Test again

## Design Patterns

### Pattern 1: High-level Guide with References

```markdown
# Data Processing

## Quick Start
Basic example here

## Advanced Features
- **Complex ops**: See [ADVANCED.md](references/ADVANCED.md)
- **API docs**: See [API.md](references/API.md)
```

### Pattern 2: Domain-Specific Organization

```
cloud-deploy/
├── SKILL.md (workflow + provider selection)
└── references/
    ├── aws.md
    ├── gcp.md
    └── azure.md
```

### Pattern 3: Conditional Details

```markdown
# Processing

## Basic Usage
Simple operations inline

**For complex cases**: See [COMPLEX.md](references/COMPLEX.md)
```

## What NOT to Include

Never create these files in skills:
- README.md (SKILL.md serves this purpose)
- INSTALLATION_GUIDE.md
- CHANGELOG.md
- Any user-facing documentation

Skills are for AI agents, not humans reading docs.

## AgentStack-Specific Guidance

When creating skills for AgentStack:

1. **Reference the specs**: Point to `/spec/*.md` files
2. **Use Go idioms**: Follow Go project conventions
3. **K8s-native**: Assume Kubernetes deployment context
4. **Multi-tenant aware**: Consider project isolation
5. **Observability built-in**: Include OpenTelemetry patterns

## Resources

- `scripts/init_skill.py` - Initialize new skills
- `references/skill-patterns.md` - Advanced design patterns
- `references/agentstack-context.md` - Platform-specific guidance
