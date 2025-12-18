# How to Claude Skills in VSCode

Publisher: Raphael MANSUY
Created Time: December 15, 2025 10:17 AM
Last edited Time: December 15, 2025 11:26 AM

Assuming intermediate audience level (familiar with VSCode and AI coding assistants), target length as short as possible while complete, prerequisites: VSCode 1.107 installed, access to Claude API via extensions like Continue or GitHub Copilot Chat, basic CLI and Markdown skills.

# Why Claude Skills in VSCode

- **Modular AI Expertise**: Packages reusable, domain-specific workflows into skills that Claude invokes dynamically, accelerating dev tasks like code reviews or commits in VSCode.
- **IDE-Native Integration**: Excels in VSCode 1.107's agent ecosystem for file ops, git interactions, and orchestration, minimizing switches from tools like Cursor.
- **Context-Efficient Extension**: Complements Claude's base capabilities with custom instructions/scripts, outperforming generic prompts without retraining.
- **Current Relevance**: Critical in 2025 with VSCode's multi-agent support and Claude's API updates, enabling parallel subtasks in evolving projects.
- **Replaces Ad-Hoc Tools**: Substitutes scattered prompts or subagents with versioned, shareable skills for team consistency.
- **Scoped Security**: Restricts tool access per skill, ideal for enterprise environments handling sensitive code.
- **Community-Driven**: Leverages Anthropic's official and user-built skills for high-value workflows, from AWS deployments to transcript analysis.

# What problems it solves (and what it doesn't)

| Good fit | Bad fit |
| --- | --- |
| Modularizing repetitive dev tasks like commit generation, PR reviews, or test writing with custom instructions/scripts. | Real-time multi-user collaboration beyond chat-based agents; use tools like Live Share instead. |
| Injecting project-specific knowledge (e.g., AWS CDK patterns) into Claude for efficient VSCode workflows. | Heavy computation like ML training; skills are lightweight, not compute-intensive apps. |
| Orchestrating subtasks in VSCode agents for debugging, file processing, or artifact building. | Non-dev domains without tool access (e.g., pure creative writing); skills depend on Claude's filesystem/tools. |

Real-world scenarios:

- **PR Review Automation**: In a GitHub repo, a skill analyzes diffs, checks security/best practices, and suggests edits—resolves inconsistent manual reviews.
- **Artifact Generation**: For UI prototypes, a skill uses React/Tailwind to build components from specs—streamlines frontend iteration in VSCode.
- **Transcript Analysis**: Processes meeting notes to extract insights/action items—boosts productivity in dev teams tracking discussions.

# Mental model / key concepts (the minimum to think correctly)

Claude Skills are self-contained folders providing specialized knowledge: core [`SKILL.md`](http://SKILL.md) (YAML metadata + Markdown instructions) defines activation triggers, with optional scripts/templates/references. VSCode/Claude scans `~/.claude/skills/` (personal) or `.claude/skills/` (project) dirs, matching queries to descriptions for on-demand loading. Skills use scoped tools (e.g., Read, Git) via `allowed-tools`, enabling agent delegation without state persistence—progressive context loading minimizes tokens.

```
Skill Activation Flow:
User Query --> Description Match? --> Load Metadata (always)
                                          |
                                          v
                                Load Instructions (if triggered)
                                          |
                                          v
                             Load Resources/Scripts (as needed) --> Execute
```

Glossary:

- **Skill**: Folder-based package with triggers for task-specific expertise.
- **Agent**: VSCode/Claude entity that discovers/invokes skills in orchestration.
- **Allowed-tools**: YAML array limiting ops (e.g., Read, Git, Bash) for security.
- **Description**: Query-matching text in [SKILL.md](http://SKILL.md) for activation.
- **Progressive Loading**: Tiered context (metadata > instructions > resources) for efficiency.

# The survival kit (actionable fastest path to proficiency)

**Prioritized checklist**:

- **Day 0**: Install VSCode 1.107, Claude extension (e.g., Continue). Enable `chat.useClaudeSkills`. Clone official repo (anthropics/skills) to `~/.claude/skills/`.
- **Week 1**: Test 3-5 community skills (e.g., code-reviewer, commit-helper). Invoke via queries like "Review this PR." Build one custom skill using skill-creator.
- **Week 2**: Integrate into projects via `.claude/skills/`. Use in background agents for async ops. Experiment with stacking (e.g., test-writer + bug-finder).

**20% of features for 80% results**: [SKILL.md](http://SKILL.md) (name/description/instructions); dir placement; natural query invocation; allowed-tools for scoping.

**Common pitfalls + avoidance**:

- Poor descriptions cause missed activations—use precise keywords (e.g., "analyze AWS CDK for cost optimization").
- Tool failures from missing scopes—list explicit allowed-tools; verify in agent settings.
- Context overload in large skills—keep instructions <2k tokens; use references for deferred loading.

**Debugging/observability tips**: Query "List available skills" in chat. Inspect session logs (Chat > Sessions). Use "Claude: Reload Skills" in Command Palette.

**Performance and security gotchas**: Progressive loading saves tokens but slows complex stacks—prioritize essential resources. Restrict Write tools in shared skills to prevent unauthorized changes.

# Progressive complexity examples (high value, minimal but real)

**Example 1: "Hello, core primitive" (smallest useful win)**

- **Problem statement**: Test skill discovery with a simple echo for setup validation.
- **Solution** (from Anthropic's official repo):

```
# ~/.claude/skills/hello-skill/[SKILL.md](http://SKILL.md)
---
name: hello-skill
description: Echo input for testing. Trigger on 'test skill activation'.
---
# Hello Skill
## Instructions
1. Reply: "Echo from skill: [user input]".
```

- **How it works**: Dir placement enables scan. Query "Test skill activation with message." Claude matches, loads, executes.
- **When to use / when not**: Quick validation; avoid for prod—lacks tools/integration.
- **One "upgrade" idea**: Add logging script for invocation tracking (reliability).

**Example 2: "Typical workflow"**

- **Problem statement**: Generate git commit messages from diffs consistently.
- **Solution** (user-built, from ComposioHQ/awesome-claude-skills):

```
# .claude/skills/commit-message-generator/[SKILL.md](http://SKILL.md)
---
name: commit-message-generator
description: Create structured commit messages from staged changes. Use for git commits.
allowed-tools: Read, Git
---
# Commit Message Generator
## Instructions
1. Fetch diff via git.
2. Format: Summary (<50 chars), body, affected files.
```

- **How it works**: In VSCode, stage files. Query "Generate commit message." Skill activates, uses tools, outputs formatted message.
- **When to use / when not**: Daily git flows; not for non-git or human-needed merges.
- **One "upgrade" idea**: Integrate template.txt for custom formats (ergonomics).

**Example 3: "Production-ish pattern"**

- **Problem statement**: Review code for errors, practices, security in PRs.
- **Solution** (community, from simonw/claude-skills):

```
# .claude/skills/code-reviewer/[SKILL.md](http://SKILL.md)
---
name: code-reviewer
description: Scan code for best practices, bugs, vulnerabilities. For PR/code quality.
allowed-tools: Read, Grep, Glob
---
# Code Reviewer
## Instructions
1. Glob sources.
2. Grep issues (e.g., security patterns).
3. Output: Checklist with lines, suggestions.
Reference: [[best-practices.md](http://best-practices.md)]([best-practices.md](http://best-practices.md))
```

- **How it works**: Attach PR context in chat. Query "Review code." Loads files, analyzes, provides feedback.
- **When to use / when not**: Team CI/CD; not for subjective design.
- **One "upgrade" idea**: Add Python script for metric calculations (testability).

**Example 4: "Advanced but common"**

- **Problem statement**: Optimize AWS CDK deployments with cost checks.
- **Solution** (high-value user skill from ComposioHQ repo):

```
# ~/.claude/skills/aws-cdk-optimizer/[SKILL.md](http://SKILL.md)
---
name: aws-cdk-optimizer
description: Generate/optimize AWS CDK stacks with cost analysis. For cloud infra.
allowed-tools: Read, Bash, Git
---
# AWS CDK Optimizer
## Instructions
1. Read CDK code.
2. Suggest optimizations (e.g., resource sizing).
3. Estimate costs via AWS CLI.
Scripts: [[cost-estimator.py](http://cost-estimator.py)](scripts/[cost-estimator.py](http://cost-estimator.py))
```

- **How it works**: In VSCode project, query "Optimize CDK stack." Executes script, outputs improved code/costs.
- **When to use / when not**: Infra provisioning; not for non-AWS.
- **One "upgrade" idea**: Add scaling patterns for high-traffic (scalability).

**Example 5: "Advanced but common"**

- **Problem statement**: Analyze meeting transcripts for insights/actions.
- **Solution** (Anthropic official-inspired, community variant):

```
# ~/.claude/skills/meeting-transcript-analyzer/[SKILL.md](http://SKILL.md)
---
name: meeting-transcript-analyzer
description: Extract key points, behaviors, actions from transcripts. For dev meetings.
allowed-tools: Read
---
# Transcript Analyzer
## Instructions
1. Parse transcript.
2. Identify: Decisions, tasks, sentiments.
3. Output: Summary, assignee list.
```

- **How it works**: Upload transcript. Query "Analyze meeting notes." Processes, generates report.
- **When to use / when not**: Post-meeting reviews; not for real-time.
- **One "upgrade" idea**: Integrate with Slack via script (ergonomics).

**Example 6: "Advanced but common"**

- **Problem statement**: Automate Playwright tests for browser UI.
- **Solution** (from awesome-claude-skills repos):

```
# .claude/skills/playwright-tester/[SKILL.md](http://SKILL.md)
---
name: playwright-tester
description: Generate/run Playwright tests for web apps. For UI validation.
allowed-tools: Bash, Write
---
# Playwright Tester
## Instructions
1. Install Playwright if needed.
2. Write tests from specs.
3. Run and report.
Scripts: [test-runner.js](scripts/test-runner.js)
```

- **How it works**: Query "Test UI with Playwright." Generates/runs tests in VSCode terminal.
- **When to use / when not**: E2E testing; not for non-web.
- **One "upgrade" idea**: Add CI integration hooks (reliability).

# Cheat sheet

- Dir setup: `mkdir -p ~/.claude/skills/my-skill` or `.claude/skills/` for projects.
- [SKILL.md](http://SKILL.md) base: `---\nname: skill-name\ndescription: Trigger phrase.\nallowed-tools: Read\n---\n# Title\n## Instructions\n1. Step...\n`.
- List skills: Chat "What skills are available?".
- Invoke: Query matching description, e.g., "Optimize AWS CDK".
- Tool scope: `allowed-tools: Read, Git, Bash`.
- Git skill: Place in repo, `git add .claude/skills/`.
- Test activation: "Use [skill-name] for [task]".
- Reload: Cmd+Shift+P > "Claude: Reload Skills".
- Debug: Refine desc with synonyms; check logs.
- Stack skills: Query implies multiple (e.g., "Review and test code").
- Official clone: `git clone [https://github.com/anthropics/skills](https://github.com/anthropics/skills) ~/.claude/skills/anthropic`.
- Community install: Clone repos like ComposioHQ/awesome-claude-skills.
- Script ref: "Execute [scripts/[helper.py](http://helper.py)] with args".
- Resource load: Link as `[[ref.md](http://ref.md)]([reference.md](http://reference.md))`.
- Bash tool: Allow Bash for CLI exec.
- VSCode enable: Settings > `chat.useClaudeSkills: true`.
- Background: Attach skills to async agents.
- Token opt: Keep desc <512 chars.
- Security: Omit Write in read-only skills.
- Update: Edit, reload session.

**If you only remember 5 things**:

- [SKILL.md](http://SKILL.md) triggers via name/desc.
- Dirs: ~/.claude/ or .claude/.
- Query: "List skills" for discovery.
- Tools: Scope with allowed-tools.
- Enable: VSCode setting `chat.useClaudeSkills`.

# Related technologies & concepts (map of the neighborhood)

- **Alternatives**: VSCode Custom Agents—pick for native YAML; less modular but deeper IDE tie-in.
- **Complements**: [Continue.dev](http://Continue.dev) extension—enhances Claude integration; Background Agents for skill async.
- **Prereqs**: Claude API key; Git for skill sharing.
- **Next steps**: Claude Code Plugins—bundle advanced capabilities; Multi-agent for delegation.

| Skill Name | Description | URL |
| --- | --- | --- |
| Skill-Creator | Built-in skill to interview users and generate new custom skills. | [https://github.com/anthropics/skills/tree/main/skill-creator](https://github.com/anthropics/skills/tree/main/skill-creator) |
| Excel-Creator | Creates structured Excel spreadsheets from data/queries. | [https://github.com/anthropics/skills/tree/main/excel-creator](https://github.com/anthropics/skills/tree/main/excel-creator) |
| PowerPoint-Creator | Generates presentation slides with layouts/content. | [https://github.com/anthropics/skills/tree/main/powerpoint-creator](https://github.com/anthropics/skills/tree/main/powerpoint-creator) |
| Word-Doc-Creator | Builds formatted Word documents. | [https://github.com/anthropics/skills/tree/main/word-doc-creator](https://github.com/anthropics/skills/tree/main/word-doc-creator) |
| PDF-Creator | Assembles PDF files from text/images. | [https://github.com/anthropics/skills/tree/main/pdf-creator](https://github.com/anthropics/skills/tree/main/pdf-creator) |
| Commit-Message-Generator | Formats git commit messages from diffs. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/commit-message-generator](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/commit-message-generator) |
| Code-Reviewer | Analyzes code for practices/bugs/security. | [https://github.com/simonw/claude-skills/tree/main/code-reviewer](https://github.com/simonw/claude-skills/tree/main/code-reviewer) |
| AWS-CDK-Optimizer | Optimizes AWS CDK stacks with cost estimates. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/aws-cdk-optimizer](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/aws-cdk-optimizer) |
| Playwright-Tester | Generates/runs browser tests with Playwright. | [https://github.com/travisvn/awesome-claude-skills/tree/main/playwright-tester](https://github.com/travisvn/awesome-claude-skills/tree/main/playwright-tester) |
| Meeting-Transcript-Analyzer | Extracts insights/actions from transcripts. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/meeting-transcript-analyzer](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/meeting-transcript-analyzer) |
| Invoice-Organizer | Processes/organizes invoices for tax prep. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/invoice-organizer](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/invoice-organizer) |
| Git-Worktree-Manager | Manages git worktrees for branching. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/git-worktree-manager](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/git-worktree-manager) |
| Content-Researcher | Researches topics with citations. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/content-researcher](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/content-researcher) |
| Frontend-Design | Builds UI artifacts with React/Tailwind. | [https://github.com/ComposioHQ/awesome-claude-skills/tree/main/frontend-design](https://github.com/ComposioHQ/awesome-claude-skills/tree/main/frontend-design) |
| Step-Functions-Visualizer | Visualizes AWS Step Functions as diagrams. | [https://github.com/industrial-ds/claude-skills/tree/main/step-functions-visualizer](https://github.com/industrial-ds/claude-skills/tree/main/step-functions-visualizer) |
| Docstring-Generator | Adds docstrings to code functions. | [https://github.com/alirezarezvani/claude-skills/tree/main/docstring-generator](https://github.com/alirezarezvani/claude-skills/tree/main/docstring-generator) |
| Bug-Finder | Scans for common bugs/patterns. | [https://github.com/brunoasm/my_claude_skills/tree/main/bug-finder](https://github.com/brunoasm/my_claude_skills/tree/main/bug-finder) |
| Test-Writer | Generates unit/integration tests. | [https://github.com/metaskills/skill-builder/tree/main/test-writer](https://github.com/metaskills/skill-builder/tree/main/test-writer) |
| VS Code 1.107 Release Notes | Official VSCode update with Claude Skills support. | [https://code.visualstudio.com/updates/v1_107](https://code.visualstudio.com/updates/v1_107) |
| Claude Skills Documentation | Anthropic's guide to skills in Claude Code/VSCode. | [https://code.claude.com/docs/en/skills](https://code.claude.com/docs/en/skills) |
| Awesome Claude Skills (ComposioHQ) | Curated 450+ skills for workflows. | [https://github.com/ComposioHQ/awesome-claude-skills](https://github.com/ComposioHQ/awesome-claude-skills) |
| Awesome Claude Skills (travisvn) | List focused on Claude Code customizations. | [https://github.com/travisvn/awesome-claude-skills](https://github.com/travisvn/awesome-claude-skills) |