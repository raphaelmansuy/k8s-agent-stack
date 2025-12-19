# Work Plan: Protocol Integration & Spec Coherence

> Comprehensive plan for A2A, AG-UI, A2UI integration and specification cleanup

**Created**: 2025-12-17  
**Status**: In Progress  
**Priority**: High

---

## Executive Summary

This work plan addresses the need to:
1. Clean leftover/duplicate specification documents
2. Integrate industry-standard protocols (A2A, AG-UI, A2UI)
3. Ensure specification coherence across all documents
4. Create the "best possible stack" for GenAI agent platform

---

## Phase 1: Cleanup & Verification ✓

### 1.1 Document Cleanup

**Files to Delete** (leftover task files, not specs):

| File | Reason |
|------|--------|
| `spec/001-consolide-doc-ensure-stack-ready.md` | Task instructions, not a spec |
| `spec/002-verify.md` | Task instructions, not a spec |
| `spec/003-api-design.md` | OLD monolithic API (superseded by 004) |

**Status**: Pending

### 1.2 Current Valid Spec Documents

| File | Content | Status |
|------|---------|--------|
| `spec/001-platform-overview.md` | Vision, mission, stack | ✓ Valid |
| `spec/002-architecture-layers.md` | 5-layer architecture | ✓ Valid |
| `spec/003-agent-lifecycle.md` | Agent types, states | ✓ Valid |
| `spec/004-api-design.md` | REST API overview | ✓ Valid, needs A2A update |
| `spec/005-data-architecture.md` | Data layer design | ✓ Valid |
| `spec/006-security-model.md` | Security architecture | ✓ Valid |
| `spec/007-observability.md` | Monitoring, logging | ✓ Valid |
| `spec/008-deployment-topologies.md` | Deployment patterns | ✓ Valid |
| `spec/009-developer-experience.md` | DX, CLI, SDK | ✓ Valid |
| `spec/api/010-multi-tenancy.md` | Multi-tenant API | ✓ Valid |
| `spec/api/011-authentication.md` | Auth API | ✓ Valid |
| `spec/api/012-agents-endpoints.md` | Agent CRUD API | ✓ Valid |
| `spec/api/013-chat-sessions.md` | Chat API | ⚠️ Needs AG-UI update |
| `spec/api/014-tools-models.md` | Tools/Models API | ✓ Valid |
| `spec/api/015-admin-endpoints.md` | Admin API | ✓ Valid |
| `spec/api/016-schemas.md` | Data schemas | ⚠️ Needs A2A schemas |
| `spec/api/README.md` | API index | ✓ Valid |

---

## Phase 2: A2A Protocol Integration

### 2.1 A2A Core Concepts to Implement

```yaml
priority_1_must_have:
  - Agent Card discovery (/.well-known/agent-card.json)
  - A2A SendMessage endpoint (/a2a/v1/message:send)
  - A2A Task object model
  - A2A streaming via SSE (/a2a/v1/message:stream)

priority_2_should_have:
  - Task lifecycle endpoints (get, list, cancel)
  - Push notification configuration
  - Multi-turn conversation (contextId)
  - Task subscription

priority_3_nice_to_have:
  - gRPC protocol binding
  - Agent Card signing (JWS)
  - Protocol extensions
```

### 2.2 Spec Documents to Create

| Document | Content | Priority |
|----------|---------|----------|
| `spec/api/017-a2a-protocol.md` | Full A2A implementation spec | High |
| `spec/api/018-agent-discovery.md` | Agent Card specification | High |

### 2.3 Spec Documents to Update

| Document | Changes Needed |
|----------|---------------|
| `spec/004-api-design.md` | Add A2A endpoints to structure |
| `spec/003-agent-lifecycle.md` | Add Agent Card to Agent CRD |
| `spec/api/016-schemas.md` | Add A2A data schemas |

---

## Phase 3: AG-UI Protocol Integration

### 3.1 AG-UI Event Types to Implement

```yaml
lifecycle_events:
  - RunStarted
  - RunFinished
  - RunError
  - StepStarted
  - StepFinished

text_events:
  - TextMessageStart    # (currently: message_start)
  - TextMessageContent  # (currently: content_delta)
  - TextMessageEnd      # (currently: message_end)

tool_events:
  - ToolCallStart       # (currently: tool_start)
  - ToolCallArgs        # (new)
  - ToolCallEnd         # (currently: tool_result)

state_events:
  - StateSnapshot       # (new - for UI sync)
  - StateDelta          # (new - for UI sync)
```

### 3.2 Migration Strategy

1. Add new event types alongside existing ones
2. Add header for format selection: `X-AgentStack-SSE-Format: agui|legacy`
3. Default to legacy for backward compatibility
4. Deprecate legacy names in v2

### 3.3 Spec Documents to Update

| Document | Changes Needed |
|----------|---------------|
| `spec/api/013-chat-sessions.md` | AG-UI event types, format header |
| `spec/004-api-design.md` | AG-UI streaming endpoint |

---

## Phase 4: A2UI Integration (Optional)

### 4.1 A2UI Scope

A2UI enables declarative UI generation for rich responses.

```yaml
components_to_support:
  - a2ui/card          # Information cards
  - a2ui/status-badge  # Status indicators
  - a2ui/progress      # Progress bars
  - a2ui/button        # Action buttons
  - a2ui/form          # Input forms
  - a2ui/table         # Data tables
  - a2ui/chart         # Visualizations
```

### 4.2 Implementation Approach

1. Define component catalog in spec
2. Agent can return A2UI in artifact parts
3. Frontend clients render from catalog
4. Non-compatible clients receive plain text fallback

### 4.3 Spec Documents to Create

| Document | Content | Priority |
|----------|---------|----------|
| `spec/api/019-a2ui-components.md` | A2UI component catalog | Medium |

---

## Phase 5: Coherence Verification

### 5.1 Cross-Document Consistency Checks

| Check | Documents | Status |
|-------|-----------|--------|
| API versions match | All API docs | Pending |
| Resource names consistent | 012, 013, 014 | Pending |
| Error codes aligned | 004, 011, API docs | Pending |
| Event types match | 004, 013 | Pending |
| Auth methods consistent | 011, all endpoints | Pending |

### 5.2 Architecture Coherence

```yaml
verify_layers:
  - Layer 3 (Cognitive) includes A2A/MCP protocols
  - Layer 4 (Interface) includes AG-UI streaming
  - API structure reflects new endpoints
  - Agent CRD includes Agent Card fields
```

---

## Implementation Timeline

```text
Week 1: Cleanup & Foundation
├── Day 1: Delete leftover files
├── Day 2: Verify all existing specs
├── Day 3: Create 017-a2a-protocol.md (structure)
├── Day 4: Create 018-agent-discovery.md
└── Day 5: Update 004-api-design.md

Week 2: Protocol Implementation
├── Day 1: Complete 017-a2a-protocol.md
├── Day 2: Update 013-chat-sessions.md (AG-UI)
├── Day 3: Update 016-schemas.md (A2A schemas)
├── Day 4: Update 003-agent-lifecycle.md
└── Day 5: Coherence verification

Week 3: Polish & Documentation
├── Day 1: Cross-reference validation
├── Day 2: Create 019-a2ui-components.md
├── Day 3: Update README files
├── Day 4: Final review
└── Day 5: Archive old process files
```

---

## Immediate Action Items

```markdown
- [x] Create design scratchpad
- [x] Create work plan
- [ ] Delete leftover spec files
- [ ] Update spec/README.md with protocol section
- [ ] Create 017-a2a-protocol.md
- [ ] Update 013-chat-sessions.md with AG-UI events
- [ ] Update 016-schemas.md with A2A schemas
- [ ] Cross-reference verification
```

---

## Success Criteria

| Criterion | Metric |
|-----------|--------|
| All specs consistent | Zero cross-reference errors |
| A2A compliant | Agent Card + core endpoints |
| AG-UI compliant | All 16 event types supported |
| Backward compatible | Existing API unchanged |
| Documentation complete | All new features documented |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking changes | Version new endpoints (/a2a/v1/) |
| Complexity increase | Clear separation of concerns |
| Adoption friction | Maintain REST API for simple cases |
| Protocol evolution | Abstract protocol adapter layer |

---

*Work Plan v1.0 - 2025-12-17*
