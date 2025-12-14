# Kagent Web UI Streaming Issue - Investigation Report

**Date**: 2025-12-14-02-45  
**Status**: ⚠️ ISSUE IDENTIFIED - Format Incompatibility

## Problem

Kagent Web UI shows agent as "Ready" but does not display streaming responses. User messages are sent but no reply appears.

## Investigation Results

### What's Working ✅

1. **Agent is running and healthy**
   - Pod: `google-adk-agent-645889f8bb-z4mj2` (1/1 Running)
   - Health endpoint: Passing
   - Service: Accessible at `google-adk-agent.kagent.svc.cluster.local:8080`
   - Agent card: Available and correctly formatted

2. **Requests are reaching the agent**
   - Kagent controller logs show: `POST /api/a2a/kagent/google-adk-agent/ => 200 OK`
   - Agent logs show: `POST / HTTP/1.1 200 OK`
   - Request/response cycle completes successfully

3. **Agent IS streaming data**
   - All events are being streamed: function calls, function responses, text tokens
   - Example from logs: "The" → " result" → " of" → " 5" → " +" → " 3" → " is" → " 8"
   - SSE format is correct: `text/event-stream` content type
   - Events formatted properly as `data: {json}\n\n`

### What's NOT Working ❌

**Root Cause**: Event Format Incompatibility

Kagent's A2A client cannot parse the streaming events because they're in **Google ADK format** instead of **A2A protocol format**.

### Error Evidence

From Kagent controller logs:
```
ERROR client/client.go:312 Error unmarshaling event for request: 
data:{"modelVersion":"gpt-4o-mini","content":{"parts":[...],"role":"model"},...},
error:failed to unmarshal event: unsupported result kind:
```

This error repeats for EVERY streaming event (25+ times per request).

### Technical Analysis

1. **Google ADK Event Format** (what we're sending):
```json
{
  "modelVersion": "gpt-4o-mini",
  "content": {"parts": [...], "role": "model"},
  "partial": true,
  "invocationId": "e-...",
  "author": "root_agent",
  "actions": {...},
  "id": "...",
  "timestamp": 1765679785.563391
}
```

2. **A2A Protocol Format** (what Kagent expects):
```json
{
  "result": {
    "kind": "...",  // Specific A2A result kind
    // A2A-specific fields
  }
}
```

3. **Why This Happens**:
   - `to_a2a()` creates A2A endpoints but doesn't handle streaming properly
   - A2A protocol has `streaming=False` hardcoded (as discovered in earlier research)
   - Our SSE middleware streams Google ADK events, not A2A events
   - Kagent's A2A client expects A2A format and fails to parse ADK format

## Solutions Explored

### ❌ Solution 1: SSE Middleware
- **Status**: Implemented but incompatible
- **Issue**: Streams ADK format events, not A2A format
- **Result**: Events stream but Kagent can't parse them

### ⚠️ Solution 2: Direct `/run_sse` Endpoint  
- **Status**: Working for direct HTTP calls
- **Issue**: Kagent Web UI doesn't use this endpoint (uses A2A proxy instead)
- **Result**: Works with curl, doesn't work with Web UI

### ❓ Solution 3: A2A Format Conversion
- **Status**: Would require deep understanding of A2A protocol
- **Complexity**: HIGH - Need to convert every ADK event to A2A format
- **Risk**: A2A protocol is experimental and format may change

## Current Status

The agent **IS working correctly** and **IS streaming**, but the format is incompatible with Kagent's A2A client parser.

## Workarounds

### Option A: Use Kagent CLI (A2A Non-Streaming)
```bash
kagent invoke -a google-adk-agent -t "What is 5 + 3?"
```
**Status**: May work for non-streaming requests

### Option B: Direct HTTP to `/run_sse`
```bash
curl -X POST http://google-adk-agent.kagent.svc.cluster.local:8080/run_sse \
  -H "Content-Type: application/json" \
  -d '{"app_name":"google-adk-agent","user_id":"test","session_id":"test-1",
       "new_message":{"role":"user","parts":[{"text":"What is 5 + 3?"}]}}'
```
**Status**: ✅ Works perfectly with proper streaming

### Option C: Wait for Google ADK A2A Streaming Support
The A2A protocol implementation in Google ADK is experimental. Proper streaming support may be added in future releases.

## Recommendations

1. **Short Term**: Document that streaming is not currently supported via Kagent Web UI due to A2A protocol limitations

2. **Medium Term**: Consider one of:
   - Implement full A2A event format conversion in middleware
   - Work with Google ADK team to add proper A2A streaming support
   - Use alternative agent frameworks that support A2A streaming natively

3. **Long Term**: Monitor Google ADK releases for A2A streaming improvements

## Files Modified

- [fast_api_app.py](../kagent-adk-agent/app/fast_api_app.py) - Removed SSE middleware (incompatible format)
- Agent now uses default `to_a2a()` behavior

## Key Learnings

1. **A2A Protocol is Experimental**: Streaming support is limited and format-specific
2. **Format Matters**: SSE content-type alone isn't enough - event structure must match
3. **Proxy Layer Complexity**: Kagent's A2A proxy adds format requirements
4. **Google ADK vs A2A**: These are different event formats - not directly compatible

## Next Steps

User should decide:
1. Accept non-streaming responses via Kagent Web UI
2. Use direct HTTP calls to `/run_sse` endpoint for streaming
3. Implement full A2A format conversion (complex, time-consuming)
4. Wait for Google ADK to improve A2A streaming support

---

**Conclusion**: The agent works correctly and streams data properly. The issue is a format incompatibility between Google ADK events and Kagent's A2A client expectations. This is a known limitation of the experimental A2A protocol implementation in Google ADK.
