# 019 - A2UI Protocol Integration

> Agent-to-UI Declarative Component Format for Generative UI

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Overview

A2UI (Agent-to-UI) is Google's protocol for declarative, security-first UI generation by AI agents. AgentStack implements A2UI to provide:

- **Secure Rendering**: Declarative data format (not executable code)
- **Component Catalog**: Pre-approved UI components only
- **Framework Agnostic**: Works with React, Angular, Vue, Lit, Flutter
- **Rich Responses**: Beyond text - cards, forms, charts, interactive elements

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    A2UI Rendering Architecture                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Agent Response                AgentStack              Frontend        │
│   (LLM generates)               (validates)             (renders)       │
│                                                                         │
│   ┌─────────────┐              ┌─────────────┐         ┌──────────────┐│
│   │ A2UI JSON   │─────────────▶│ Component   │────────▶│ Component   ││
│   │ Declarative │              │ Validator   │         │ Renderer    ││
│   │ Format      │              │ (sandbox)   │         │ (catalog)   ││
│   └─────────────┘              └─────────────┘         └──────────────┘│
│                                                                         │
│   Security: No executable code passes through - only data structures   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Concepts

### 2.1 Design Principles

| Principle | Description |
|-----------|-------------|
| **Declarative** | UI defined as data, not code |
| **Secure** | LLM cannot inject executable code |
| **Catalog-based** | Components from approved registry only |
| **Framework-agnostic** | Same spec renders in any UI framework |
| **Incremental** | Supports partial updates and diffs |

### 2.2 Protocol Relationship

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Protocol Stack Integration                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   A2UI = What to render (UI specification)                              │
│   AG-UI = How to stream (transport/events)                              │
│   A2A = Agent interop (task/message format)                             │
│                                                                         │
│   Example Flow:                                                         │
│   1. User asks: "Show me my order status"                               │
│   2. Agent generates A2UI component spec                                │
│   3. A2UI spec embedded in AG-UI TextMessageContent event               │
│   4. Frontend renders A2UI component from catalog                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Component Catalog

### 3.1 Core Components

| Component | Description | Use Case |
|-----------|-------------|----------|
| `a2ui/card` | Container with header | Info display |
| `a2ui/text` | Rich text block | Paragraphs, markdown |
| `a2ui/status-badge` | Status indicator | Progress, state |
| `a2ui/progress` | Progress bar | Completion tracking |
| `a2ui/button` | Interactive button | Actions |
| `a2ui/link` | Hyperlink | Navigation |
| `a2ui/image` | Image display | Media |
| `a2ui/list` | Ordered/unordered list | Collections |
| `a2ui/table` | Data table | Tabular data |
| `a2ui/form` | Input form | Data collection |
| `a2ui/input` | Text input field | User input |
| `a2ui/select` | Dropdown selector | Choices |
| `a2ui/checkbox` | Checkbox input | Boolean options |
| `a2ui/chart` | Visualization | Data charts |
| `a2ui/code` | Code block | Source display |
| `a2ui/alert` | Alert message | Notifications |

### 3.2 Layout Components

| Component | Description | Use Case |
|-----------|-------------|----------|
| `a2ui/container` | Generic container | Grouping |
| `a2ui/row` | Horizontal layout | Inline elements |
| `a2ui/column` | Vertical layout | Stacked elements |
| `a2ui/grid` | Grid layout | Complex layouts |
| `a2ui/tabs` | Tab navigation | Multiple views |
| `a2ui/accordion` | Collapsible sections | Dense content |

---

## 4. Component Specifications

### 4.1 Base Component Schema

```yaml
# Every A2UI component follows this structure
{
  "type": "a2ui/{component-name}",
  "id": "unique-id",                    # Optional, for updates
  "props": {
    # Component-specific properties
  },
  "children": [                          # Optional, nested components
    # Child components
  ],
  "actions": {                           # Optional, interaction handlers
    "onClick": {"type": "action-id", "payload": {...}}
  }
}
```

### 4.2 Card Component

```yaml
{
  "type": "a2ui/card",
  "props": {
    "title": "Order Status",
    "subtitle": "Order #12345",
    "variant": "outlined",              # outlined | elevated | filled
    "header": {                         # Optional header customization
      "icon": "shopping-cart",
      "action": {"type": "dismiss"}
    }
  },
  "children": [
    # Nested components
  ]
}
```

### 4.3 Status Badge Component

```yaml
{
  "type": "a2ui/status-badge",
  "props": {
    "status": "success",                # success | warning | error | info | pending
    "label": "Shipped",
    "icon": "check-circle",             # Optional icon
    "size": "medium"                    # small | medium | large
  }
}
```

### 4.4 Progress Component

```yaml
{
  "type": "a2ui/progress",
  "props": {
    "value": 75,
    "max": 100,
    "label": "Delivery Progress",
    "showValue": true,
    "variant": "linear",                # linear | circular
    "color": "primary"                  # primary | secondary | success | warning
  }
}
```

### 4.5 Button Component

```yaml
{
  "type": "a2ui/button",
  "props": {
    "label": "Track Package",
    "variant": "primary",               # primary | secondary | text | outlined
    "size": "medium",                   # small | medium | large
    "icon": "location-on",
    "disabled": false,
    "loading": false
  },
  "actions": {
    "onClick": {
      "type": "navigate",
      "payload": {"url": "/tracking/12345"}
    }
  }
}
```

### 4.6 Form Component

```yaml
{
  "type": "a2ui/form",
  "id": "contact-form",
  "props": {
    "title": "Contact Information",
    "submitLabel": "Save"
  },
  "children": [
    {
      "type": "a2ui/input",
      "props": {
        "name": "email",
        "label": "Email Address",
        "type": "email",
        "required": true,
        "placeholder": "you@example.com"
      }
    },
    {
      "type": "a2ui/input",
      "props": {
        "name": "phone",
        "label": "Phone Number",
        "type": "tel",
        "required": false
      }
    },
    {
      "type": "a2ui/select",
      "props": {
        "name": "country",
        "label": "Country",
        "options": [
          {"value": "us", "label": "United States"},
          {"value": "uk", "label": "United Kingdom"},
          {"value": "jp", "label": "Japan"}
        ]
      }
    }
  ],
  "actions": {
    "onSubmit": {
      "type": "form-submit",
      "payload": {"formId": "contact-form"}
    }
  }
}
```

### 4.7 Table Component

```yaml
{
  "type": "a2ui/table",
  "props": {
    "title": "Recent Orders",
    "columns": [
      {"key": "id", "label": "Order ID", "width": "20%"},
      {"key": "date", "label": "Date", "width": "25%"},
      {"key": "status", "label": "Status", "width": "20%"},
      {"key": "total", "label": "Total", "width": "20%", "align": "right"},
      {"key": "actions", "label": "", "width": "15%"}
    ],
    "rows": [
      {
        "id": "#12345",
        "date": "2025-01-10",
        "status": {"type": "a2ui/status-badge", "props": {"status": "success", "label": "Delivered"}},
        "total": "$129.99",
        "actions": {"type": "a2ui/button", "props": {"label": "View", "variant": "text", "size": "small"}}
      },
      {
        "id": "#12346",
        "date": "2025-01-12",
        "status": {"type": "a2ui/status-badge", "props": {"status": "pending", "label": "Shipping"}},
        "total": "$89.50",
        "actions": {"type": "a2ui/button", "props": {"label": "Track", "variant": "text", "size": "small"}}
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 10,
      "total": 47
    }
  }
}
```

### 4.8 Chart Component

```yaml
{
  "type": "a2ui/chart",
  "props": {
    "title": "Sales Trend",
    "chartType": "line",                # line | bar | pie | area
    "data": {
      "labels": ["Jan", "Feb", "Mar", "Apr", "May"],
      "datasets": [
        {
          "label": "Revenue",
          "data": [12000, 15000, 13500, 18000, 21000],
          "color": "#4285f4"
        },
        {
          "label": "Expenses",
          "data": [8000, 9000, 8500, 10000, 11000],
          "color": "#ea4335"
        }
      ]
    },
    "options": {
      "showLegend": true,
      "showGrid": true,
      "animate": true
    }
  }
}
```

---

## 5. Transport Integration

### 5.1 A2UI in AG-UI Events

A2UI components are transported via AG-UI events:

```yaml
# A2UI content in TextMessageContent event
event: TextMessageContent
data: {
  "type": "TextMessageContent",
  "messageId": "msg_xyz",
  "contentType": "a2ui",                # Indicates A2UI payload
  "delta": {
    "type": "a2ui/card",
    "props": {
      "title": "Order Status",
      "subtitle": "Order #12345"
    },
    "children": [
      {
        "type": "a2ui/status-badge",
        "props": {"status": "success", "label": "Shipped"}
      }
    ]
  }
}
```

### 5.2 A2UI in A2A Artifacts

A2UI components as A2A task artifacts:

```yaml
# A2A response with A2UI artifact
{
  "task": {
    "id": "task_abc",
    "status": {"state": "completed"},
    "artifacts": [
      {
        "artifactId": "art_1",
        "name": "Order Status UI",
        "parts": [
          {
            "data": {
              "mediaType": "application/a2ui+json",
              "data": {
                "type": "a2ui/card",
                "props": {"title": "Order Status"},
                "children": [...]
              }
            }
          }
        ]
      }
    ]
  }
}
```

### 5.3 A2UI in REST Responses

A2UI in standard chat responses:

```yaml
POST /v1/agents/{agentId}/chat

Response:
{
  "message": {
    "id": "msg_abc",
    "role": "assistant",
    "content": [
      {
        "type": "text",
        "text": "Here's your order status:"
      },
      {
        "type": "a2ui",
        "component": {
          "type": "a2ui/card",
          "props": {"title": "Order #12345"},
          "children": [...]
        }
      }
    ]
  }
}
```

---

## 6. Actions and Interactivity

### 6.1 Action Types

| Action Type | Description | Payload |
|-------------|-------------|---------|
| `navigate` | Navigate to URL | `{url: string}` |
| `submit` | Submit form data | `{formId: string}` |
| `dismiss` | Close/dismiss component | `{}` |
| `expand` | Expand collapsed content | `{targetId: string}` |
| `copy` | Copy to clipboard | `{text: string}` |
| `download` | Download file | `{url: string, filename: string}` |
| `agent-action` | Send action to agent | `{action: string, data: object}` |

### 6.2 Agent Action Handler

When a user interacts with an A2UI component:

```yaml
# User clicks button with agent-action
{
  "type": "a2ui/button",
  "props": {"label": "Confirm Order"},
  "actions": {
    "onClick": {
      "type": "agent-action",
      "payload": {
        "action": "confirm-order",
        "data": {"orderId": "12345"}
      }
    }
  }
}

# Action sent back to agent
POST /agui/v1/actions
{
  "runId": "run_abc",
  "action": "agent-action",
  "data": {
    "action": "confirm-order",
    "data": {"orderId": "12345"}
  }
}
```

---

## 7. Validation and Security

### 7.1 Component Validation

AgentStack validates all A2UI components before delivery:

```yaml
# Validation Rules
1. Component type must be in allowed catalog
2. Props must match component schema
3. Children must be valid components
4. Actions must use allowed action types
5. No script or executable content
6. No external resource references (images must be data URIs or approved CDN)
```

### 7.2 Security Model

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    A2UI Security Model                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ❌ BLOCKED                        ✅ ALLOWED                          │
│   ─────────                         ───────────                         │
│   • JavaScript execution            • Declarative component specs       │
│   • Inline scripts                  • Pre-defined action types          │
│   • External URLs (untrusted)       • Approved CDN resources            │
│   • Arbitrary HTML                  • Catalog components only           │
│   • CSS injection                   • Theme-scoped styling              │
│   • iframes                         • Data URIs for images              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 7.3 Validation Endpoint

```yaml
# Validate A2UI component (optional, for debugging)
POST /v1/a2ui/validate
Content-Type: application/json

Request:
{
  "component": {
    "type": "a2ui/card",
    "props": {...}
  }
}

Response: 200 OK
{
  "valid": true,
  "warnings": []
}

# Invalid component
Response: 200 OK
{
  "valid": false,
  "errors": [
    {"path": "$.children[0].type", "message": "Unknown component type: a2ui/malicious"}
  ]
}
```

---

## 8. Incremental Updates

### 8.1 Component Updates

A2UI supports partial updates using component IDs:

```yaml
# Initial render
event: TextMessageContent
data: {
  "contentType": "a2ui",
  "delta": {
    "type": "a2ui/progress",
    "id": "upload-progress",
    "props": {"value": 0, "max": 100, "label": "Uploading..."}
  }
}

# Update progress (partial)
event: TextMessageContent
data: {
  "contentType": "a2ui-update",
  "delta": {
    "id": "upload-progress",
    "patch": {"props.value": 50}
  }
}

# Final update
event: TextMessageContent
data: {
  "contentType": "a2ui-update",
  "delta": {
    "id": "upload-progress",
    "patch": {"props.value": 100, "props.label": "Complete!"}
  }
}
```

---

## 9. Frontend Implementation

### 9.1 React Renderer Example

```typescript
// A2UI React Renderer
import { A2UIComponent } from '@agentstack/a2ui-react';

function AgentMessage({ content }) {
  if (content.type === 'a2ui') {
    return <A2UIComponent spec={content.component} />;
  }
  return <TextContent text={content.text} />;
}

// Component Registry
const componentRegistry = {
  'a2ui/card': CardComponent,
  'a2ui/button': ButtonComponent,
  'a2ui/status-badge': StatusBadgeComponent,
  'a2ui/progress': ProgressComponent,
  // ... more components
};
```

### 9.2 Action Handling

```typescript
// A2UI Action Handler
function handleA2UIAction(action: A2UIAction) {
  switch (action.type) {
    case 'navigate':
      router.push(action.payload.url);
      break;
    case 'agent-action':
      sendToAgent(action.payload);
      break;
    case 'copy':
      navigator.clipboard.writeText(action.payload.text);
      break;
    // ... more action types
  }
}
```

---

## 10. Agent Integration

### 10.1 Generating A2UI from Agent

```python
# Python Agent SDK - A2UI generation
from agentstack.a2ui import Card, StatusBadge, Button, Progress

def generate_order_status_ui(order):
    return Card(
        title="Order Status",
        subtitle=f"Order #{order.id}",
        children=[
            StatusBadge(
                status="success" if order.shipped else "pending",
                label=order.status_text
            ),
            Progress(
                value=order.progress_pct,
                max=100,
                label="Delivery Progress"
            ),
            Button(
                label="Track Package",
                variant="primary",
                action={"type": "navigate", "payload": {"url": f"/track/{order.id}"}}
            )
        ]
    )
```

### 10.2 LLM Prompt for A2UI

```yaml
# System prompt for A2UI generation
You can generate rich UI responses using A2UI components. When the user 
asks for visual information (charts, forms, status displays), generate 
A2UI JSON.

Available components:
- a2ui/card: Container with title
- a2ui/button: Interactive button
- a2ui/status-badge: Status indicator (success/warning/error/pending)
- a2ui/progress: Progress bar
- a2ui/table: Data table
- a2ui/chart: Visualization (line/bar/pie)
- a2ui/form: Input form with fields

Example response format:
{
  "text": "Here's your order status:",
  "ui": {
    "type": "a2ui/card",
    "props": {"title": "Order #12345"},
    "children": [...]
  }
}
```

---

## 11. Configuration

### 11.1 Tenant A2UI Settings

```yaml
# Per-tenant A2UI configuration
POST /v1/admin/projects/{projectId}/settings
{
  "a2ui": {
    "enabled": true,
    "allowedComponents": [
      "a2ui/card",
      "a2ui/button", 
      "a2ui/text",
      "a2ui/status-badge",
      "a2ui/progress",
      "a2ui/table"
    ],
    "blockedComponents": [
      "a2ui/form"           # Disable forms for this tenant
    ],
    "customTheme": {
      "primaryColor": "#1a73e8",
      "fontFamily": "Inter"
    }
  }
}
```

### 11.2 Agent A2UI Capability

```yaml
# Agent definition with A2UI capability
apiVersion: kagent.dev/v1alpha1
kind: Agent
metadata:
  name: customer-service
spec:
  capabilities:
    a2ui:
      enabled: true
      preferredComponents:
        - a2ui/card
        - a2ui/status-badge
        - a2ui/button
```

---

## 12. References

- [A2UI Protocol](https://google.github.io/A2A/a2ui/)
- [AG-UI Integration](018-agui-protocol.md)
- [A2A Protocol Integration](017-a2a-protocol.md)
- [Chat Sessions API](013-chat-sessions.md)

---

*End of A2UI Protocol Specification*
