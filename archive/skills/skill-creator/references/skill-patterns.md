# Skill Design Patterns

Advanced patterns for creating effective skills.

## Pattern: Workflow Decision Tree

For skills with conditional branches:

```markdown
## Workflow Decision Tree

┌─────────────────────────────────────────┐
│           User Request                   │
└─────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
   Read-only?              Modify?
        │                       │
        ▼                       ▼
   Path A                  Path B
```

## Pattern: Progressive Loading

For large reference docs:

```markdown
## Feature X

Basic usage inline.

**Detailed docs**: See [FEATURE_X.md](references/FEATURE_X.md)
- Sections: Configuration, Advanced Usage, Troubleshooting
- Use grep pattern: `## Configuration` to find setup
```

## Pattern: Script-First

For deterministic operations:

```markdown
## Rotate PDF

Use the rotation script directly:

\`\`\`bash
python scripts/rotate_pdf.py --input file.pdf --degrees 90
\`\`\`

Script handles all edge cases. Only read script source if patching needed.
```

## Pattern: Template-Based

For consistent output generation:

```markdown
## Generate Component

1. Copy template from `assets/component-template/`
2. Modify according to requirements
3. Template includes:
   - Base structure
   - Common patterns
   - Test scaffolding
```

## Pattern: Multi-Framework Support

For skills supporting multiple frameworks:

```markdown
## Framework Selection

Determine framework from project context:
- `go.mod` present → Go patterns
- `pyproject.toml` present → Python patterns
- `package.json` present → Node.js patterns

See framework-specific guidance in:
- [references/go.md](references/go.md)
- [references/python.md](references/python.md)
- [references/nodejs.md](references/nodejs.md)
```

## Anti-Patterns to Avoid

### ❌ Over-explaining Basics

```markdown
# Bad: Explaining what everyone knows
## What is an API?
An API is an Application Programming Interface...
```

### ❌ Duplicate Information

```markdown
# Bad: Same info in SKILL.md and references
## Configuration (SKILL.md)
Set PORT=8080...

# Same in references/config.md
Set PORT=8080...
```

### ❌ Deeply Nested References

```markdown
# Bad: References pointing to other references
See [A.md] → which says see [B.md] → which says see [C.md]
```

### ❌ Vague Descriptions

```yaml
# Bad: Doesn't tell Claude when to use it
description: Helps with coding tasks
```

## Best Practices

1. **One purpose per skill**: Don't overload skills
2. **Clear activation**: Description explicitly states triggers
3. **Minimal context**: Only what Claude doesn't know
4. **Tested workflows**: Try before publishing
5. **Iterative improvement**: Update based on real usage
