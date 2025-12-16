

## Task: Procedure Verification for K8S / OrbStack Environment

### Objective

Verify the **correctness, completeness, and reliability** of the provided procedure by executing it end‑to‑end in a **clean K8S / OrbStack environment**. The goal is to ensure the procedure consistently results in a **fully functional environment** with **kagent and all related components running as expected**, and that the documentation accurately reflects reality.

---

### Execution Steps

1. **Environment Reset**
   - Fully clean and reset the local **Kubernetes + OrbStack** environment.
   - Confirm no residual clusters, contexts, images, volumes, or configurations remain.

2. **Procedure Execution**
   - Follow the procedure **exactly as written**, without skipping or reordering steps.
   - Do not apply fixes or workarounds until issues are documented.

3. **Observation & Validation**
   - Verify:
     - Kubernetes cluster health
     - OrbStack integration
     - kagent deployment and readiness
     - All dependent components are running and accessible
   - Validate expected outputs, logs, and behaviors at each stage.

---

### Documentation & Reporting (Required)

All findings must be recorded in the `./process` directory.

#### 1. Discrepancies & Issues
- Document any:
  - Errors
  - Missing steps
  - Incorrect assumptions
  - Ambiguous or misleading instructions
  - Unexpected behavior
- Include:
  - Exact commands run
  - Error messages and logs
  - Environment details (versions, configs)

#### 2. Fixes
- **Minor issues**
  - Apply fixes where safe and obvious
  - Clearly document:
    - What was changed
    - Why it was necessary
    - Whether the procedure should be updated
- **Major issues**
  - Do **not** workaround silently
  - Document thoroughly:
    - Root cause (if known)
    - Impact
    - Why it blocks completion
    - Suggested remediation

#### 3. Success Confirmation
- If the procedure completes without issues:
  - Explicitly document successful completion
  - Include verification evidence (status checks, commands, outputs)

---

### Final Deliverables

- ✅ A verified, fully functional **K8S / OrbStack environment**
- ✅ **kagent and related components** running and healthy
- ✅ Clear, accurate, and reproducible documentation
- ✅ Actionable notes for improving the procedure where needed

---

### Quality Bar

- Documentation must be:
  - Precise
  - Reproducible
  - Suitable for a new user following it for the first time
- Prefer clarity and correctness over brevity
- No undocumented assumptions or implicit steps

