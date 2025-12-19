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

import os
import json
import uuid
import warnings
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Union
from dataclasses import dataclass, field, asdict

import google.auth
from google.adk.a2a.utils.agent_to_a2a import to_a2a
from google.adk.runners import Runner
from google.adk.artifacts.in_memory_artifact_service import InMemoryArtifactService
from google.adk.sessions.in_memory_session_service import InMemorySessionService
from google.adk.memory.in_memory_memory_service import InMemoryMemoryService
from google.adk.auth.credential_service.in_memory_credential_service import InMemoryCredentialService
from google.adk.agents.run_config import RunConfig, StreamingMode
from google.genai import types
from google.cloud import logging as google_cloud_logging
from starlette.requests import Request
from starlette.responses import StreamingResponse, JSONResponse
from starlette.routing import Route
from pydantic import BaseModel

from app.agent import root_agent
from app.app_utils.telemetry import setup_telemetry


# ============================================================================
# A2A Protocol Types (matching Kagent's expected format)
# ============================================================================

@dataclass
class TextPart:
    """A2A TextPart - text content in a message."""
    text: str
    kind: str = "text"
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {"kind": self.kind, "text": self.text}
        if self.metadata:
            result["metadata"] = self.metadata
        return result


@dataclass
class DataPart:
    """A2A DataPart - structured data (e.g., function calls/responses)."""
    data: Dict[str, Any]
    kind: str = "data"
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {"kind": self.kind, "data": self.data}
        if self.metadata:
            result["metadata"] = self.metadata
        return result


@dataclass 
class Part:
    """A2A Part wrapper - wraps TextPart or DataPart."""
    root: Union[TextPart, DataPart]
    
    def to_dict(self) -> Dict[str, Any]:
        return self.root.to_dict()


@dataclass
class Message:
    """A2A Message - contains parts and metadata."""
    message_id: str
    role: str  # "user" or "agent"
    parts: List[Part]
    kind: str = "message"
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "kind": self.kind,
            "messageId": self.message_id,
            "role": self.role,
            "parts": [p.to_dict() for p in self.parts]
        }
        if self.metadata:
            result["metadata"] = self.metadata
        return result


@dataclass
class TaskStatus:
    """A2A TaskStatus - current state of a task."""
    state: str  # "submitted", "working", "completed", "failed", etc.
    timestamp: str  # ISO 8601 format
    message: Optional[Message] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "state": self.state,
            "timestamp": self.timestamp
        }
        if self.message:
            result["message"] = self.message.to_dict()
        return result


@dataclass
class TaskStatusUpdateEvent:
    """A2A TaskStatusUpdateEvent - streaming status update."""
    task_id: str
    context_id: str
    status: TaskStatus
    kind: str = "status-update"
    final: bool = False
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "kind": self.kind,
            "taskId": self.task_id,
            "contextId": self.context_id,
            "status": self.status.to_dict(),
            "final": self.final
        }
        if self.metadata:
            result["metadata"] = self.metadata
        return result


@dataclass
class Artifact:
    """A2A Artifact - output data from an agent."""
    artifact_id: str
    parts: List[Part]
    name: Optional[str] = None
    description: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "artifactId": self.artifact_id,
            "parts": [p.to_dict() for p in self.parts]
        }
        if self.name:
            result["name"] = self.name
        if self.description:
            result["description"] = self.description
        if self.metadata:
            result["metadata"] = self.metadata
        return result


@dataclass
class TaskArtifactUpdateEvent:
    """A2A TaskArtifactUpdateEvent - artifact output from agent."""
    task_id: str
    context_id: str
    artifact: Artifact
    kind: str = "artifact-update"
    append: bool = False
    last_chunk: bool = False
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            "kind": self.kind,
            "taskId": self.task_id,
            "contextId": self.context_id,
            "artifact": self.artifact.to_dict(),
            "append": self.append,
            "lastChunk": self.last_chunk
        }
        if self.metadata:
            result["metadata"] = self.metadata
        return result


# ============================================================================
# ADK to A2A Event Converter
# ============================================================================

def get_kagent_metadata_key(key: str) -> str:
    """Generate a kagent-prefixed metadata key."""
    return f"kagent_{key}"


def convert_adk_event_to_a2a_events(
    adk_event: Any,
    task_id: str,
    context_id: str,
    app_name: str = "google-adk-agent",
    session_id: Optional[str] = None,
) -> List[Union[TaskStatusUpdateEvent, TaskArtifactUpdateEvent]]:
    """
    Convert a Google ADK Event to A2A protocol events.
    
    This maps the ADK event structure to A2A's TaskStatusUpdateEvent format
    that Kagent expects.
    
    Args:
        adk_event: The ADK event from runner.run_async()
        task_id: The A2A task ID
        context_id: The A2A context ID
        app_name: Application name for metadata
        session_id: Session ID for metadata
        
    Returns:
        List of A2A events (usually one TaskStatusUpdateEvent per ADK event)
    """
    a2a_events = []
    now_iso = datetime.now(timezone.utc).isoformat()
    
    # Build common metadata
    metadata = {
        get_kagent_metadata_key("app_name"): app_name,
    }
    if session_id:
        metadata[get_kagent_metadata_key("session_id")] = session_id
    
    # Extract text content from ADK event
    content = getattr(adk_event, 'content', None)
    partial = getattr(adk_event, 'partial', True)
    author = getattr(adk_event, 'author', 'agent')
    invocation_id = getattr(adk_event, 'invocation_id', None)
    
    if invocation_id:
        metadata[get_kagent_metadata_key("invocation_id")] = invocation_id
    if author:
        metadata[get_kagent_metadata_key("author")] = author
    
    # Process content to A2A parts
    a2a_parts = []
    
    if content:
        parts = getattr(content, 'parts', None)
        if parts:
            for part in parts:
                # Handle different part types
                if hasattr(part, 'text') and part.text:
                    a2a_parts.append(Part(root=TextPart(text=part.text)))
                elif hasattr(part, 'function_call') and part.function_call:
                    # Function call (tool invocation)
                    fc = part.function_call
                    a2a_parts.append(Part(root=DataPart(
                        data={
                            "id": getattr(fc, 'id', str(uuid.uuid4())),
                            "name": getattr(fc, 'name', 'unknown'),
                            "args": dict(getattr(fc, 'args', {})) if hasattr(fc, 'args') else {},
                        },
                        metadata={
                            get_kagent_metadata_key("type"): "function_call",
                        }
                    )))
                elif hasattr(part, 'function_response') and part.function_response:
                    # Function response (tool result)
                    fr = part.function_response
                    a2a_parts.append(Part(root=DataPart(
                        data={
                            "id": getattr(fr, 'id', str(uuid.uuid4())),
                            "name": getattr(fr, 'name', 'unknown'),
                            "response": getattr(fr, 'response', {}),
                        },
                        metadata={
                            get_kagent_metadata_key("type"): "function_response",
                        }
                    )))
    
    # Create A2A message if we have parts
    if a2a_parts:
        role = "agent" if author != "user" else "user"
        a2a_message = Message(
            message_id=str(uuid.uuid4()),
            role=role,
            parts=a2a_parts,
            metadata=metadata
        )
        
        # Determine state based on partial flag
        state = "working" if partial else "completed"
        is_final = not partial
        
        # Create TaskStatusUpdateEvent
        status_event = TaskStatusUpdateEvent(
            task_id=task_id,
            context_id=context_id,
            status=TaskStatus(
                state=state,
                timestamp=now_iso,
                message=a2a_message
            ),
            final=is_final,
            metadata=metadata
        )
        a2a_events.append(status_event)
    
    return a2a_events


def create_final_artifact_event(
    accumulated_text: str,
    task_id: str,
    context_id: str,
    app_name: str = "google-adk-agent",
) -> TaskArtifactUpdateEvent:
    """
    Create a final TaskArtifactUpdateEvent with accumulated text.
    
    This is sent at the end of streaming to provide the complete response
    as an artifact.
    """
    return TaskArtifactUpdateEvent(
        task_id=task_id,
        context_id=context_id,
        artifact=Artifact(
            artifact_id=str(uuid.uuid4()),
            parts=[Part(root=TextPart(text=accumulated_text))],
            name="agent_response",
            description="Complete response from agent"
        ),
        append=False,
        last_chunk=True,
        metadata={
            get_kagent_metadata_key("app_name"): app_name,
        }
    )


def create_final_status_event(
    task_id: str,
    context_id: str,
) -> TaskStatusUpdateEvent:
    """Create a final completed status event."""
    return TaskStatusUpdateEvent(
        task_id=task_id,
        context_id=context_id,
        status=TaskStatus(
            state="completed",
            timestamp=datetime.now(timezone.utc).isoformat()
        ),
        final=True
    )

# Suppress experimental warnings for cleaner logs
warnings.filterwarnings('ignore', message='.*EXPERIMENTAL.*')

setup_telemetry()

# Try to get project_id from auth or fallback to environment variable
try:
    _, project_id = google.auth.default()
except google.auth.exceptions.DefaultCredentialsError:
    project_id = os.environ.get("GOOGLE_CLOUD_PROJECT", "demo-project")

# Only initialize Cloud Logging if we have valid credentials
try:
    logging_client = google_cloud_logging.Client()
    logger = logging_client.logger(__name__)
    has_cloud_logging = True
except Exception:
    # Fallback to standard Python logging if Cloud Logging is not available
    import logging
    logger = logging.getLogger(__name__)
    has_cloud_logging = False

# Create a Runner instance for A2A SSE streaming
runner = Runner(
    app_name="google-adk-agent",
    agent=root_agent,
    artifact_service=InMemoryArtifactService(),
    session_service=InMemorySessionService(),
    memory_service=InMemoryMemoryService(),
    credential_service=InMemoryCredentialService(),
)


# ============================================================================
# A2A App with ASGI Middleware for Streaming Format Conversion
# ============================================================================

async def _handle_message_send_streaming(body_json: dict):
    """Handle message/send with SSE streaming and A2A format conversion."""
    params = body_json.get("params", {})
    message = params.get("message", {})
    
    context_id = message.get("contextId", str(uuid.uuid4()))
    task_id = str(uuid.uuid4())
    request_id = body_json.get("id", str(uuid.uuid4()))
    
    # Extract text from message parts
    user_text = ""
    message_parts = message.get("parts", [])
    for part in message_parts:
        if part.get("kind") == "text":
            user_text += part.get("text", "")
    
    if not user_text:
        return JSONResponse(
            {"jsonrpc": "2.0", "error": {"code": -32602, "message": "No text content in message"}, "id": request_id},
            status_code=200
        )
    
    # Create ADK content from user message
    new_message_content = types.Content(
        role="user",
        parts=[types.Part.from_text(text=user_text)]
    )
    
    session_id = context_id
    user_id = "default-user"
    
    # Get or create session
    session = await runner.session_service.get_session(
        app_name=runner.app_name,
        user_id=user_id,
        session_id=session_id
    )
    
    if not session:
        session = await runner.session_service.create_session(
            app_name=runner.app_name,
            user_id=user_id,
            session_id=session_id
        )
    
    # Generate streaming response with A2A format
    async def a2a_event_generator():
        # Send initial "working" status to keep connection open
        initial_status = TaskStatusUpdateEvent(
            task_id=task_id,
            context_id=context_id,
            status=TaskStatus(
                state="working",
                timestamp=datetime.now(timezone.utc).isoformat(),
            ),
            final=False
        )
        sse_response = {
            "jsonrpc": "2.0",
            "id": request_id,
            "result": initial_status.to_dict()
        }
        yield f"data: {json.dumps(sse_response)}\n\n"
        
        try:
            run_config = RunConfig(streaming_mode=StreamingMode.SSE)
            
            try:
                async_gen = runner.run_async(
                    user_id=user_id,
                    session_id=session_id,
                    new_message=new_message_content,
                    run_config=run_config,
                )
                
                async for adk_event in async_gen:
                    a2a_events = convert_adk_event_to_a2a_events(
                        adk_event,
                        task_id=task_id,
                        context_id=context_id,
                        app_name=runner.app_name,
                        session_id=session_id,
                    )
                    
                    for a2a_event in a2a_events:
                        sse_response = {
                            "jsonrpc": "2.0",
                            "id": request_id,
                            "result": a2a_event.to_dict()
                        }
                        yield f"data: {json.dumps(sse_response)}\n\n"
            except Exception as inner_e:
                import traceback
                traceback.print_exc()
                raise
            
            # Note: We don't send a separate TaskArtifactUpdateEvent because
            # the final TaskStatusUpdateEvent with final=True already contains
            # the complete message. Sending both would cause duplicate messages
            # in the Kagent Web UI.
            
        except Exception as e:
            import traceback
            traceback.print_exc()
            
            error_status = TaskStatusUpdateEvent(
                task_id=task_id,
                context_id=context_id,
                status=TaskStatus(
                    state="failed",
                    timestamp=datetime.now(timezone.utc).isoformat(),
                    message=Message(
                        message_id=str(uuid.uuid4()),
                        role="agent",
                        parts=[Part(root=TextPart(text=f"Error: {e!s}"))]
                    )
                ),
                final=True
            )
            sse_response = {
                "jsonrpc": "2.0",
                "id": request_id,
                "result": error_status.to_dict()
            }
            yield f"data: {json.dumps(sse_response)}\n\n"
    
    return StreamingResponse(
        a2a_event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
            "Connection": "keep-alive",
        }
    )


class A2AStreamingMiddleware:
    """ASGI Middleware to intercept message/send and message/stream for custom streaming."""
    
    def __init__(self, app):
        self.app = app
    
    async def __call__(self, scope, receive, send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        
        # Only intercept POST requests to /
        if scope["method"] != "POST" or scope["path"] != "/":
            await self.app(scope, receive, send)
            return
        
        # Read the request body
        body_chunks = []
        while True:
            message = await receive()
            body_chunks.append(message.get("body", b""))
            if not message.get("more_body", False):
                break
        body = b"".join(body_chunks)
        
        # Parse the JSON to check method
        try:
            body_json = json.loads(body) if body else {}
            method = body_json.get("method", "")
        except json.JSONDecodeError:
            method = ""
        
        # Check Accept header for streaming requests
        headers = dict(scope.get("headers", []))
        accept = headers.get(b"accept", b"").decode("utf-8", errors="ignore")
        wants_streaming = "text/event-stream" in accept
        
        # Create receive wrapper that replays the body
        body_sent = False
        async def receive_wrapper():
            nonlocal body_sent
            if not body_sent:
                body_sent = True
                return {"type": "http.request", "body": body, "more_body": False}
            # Wait for disconnect (this shouldn't happen in practice)
            return {"type": "http.disconnect"}
        
        # Only intercept message/send and message/stream when SSE is requested
        if method in ("message/send", "message/stream") and wants_streaming:
            try:
                # Send SSE headers directly
                await send({
                    "type": "http.response.start",
                    "status": 200,
                    "headers": [
                        [b"content-type", b"text/event-stream"],
                        [b"cache-control", b"no-cache"],
                        [b"connection", b"keep-alive"],
                        [b"x-accel-buffering", b"no"],
                    ],
                })
                
                # Get the async generator from the handler
                response = await _handle_message_send_streaming(body_json)
                
                if hasattr(response, 'body_iterator'):
                    # It's a StreamingResponse, iterate over its body
                    async for chunk in response.body_iterator:
                        await send({
                            "type": "http.response.body",
                            "body": chunk.encode() if isinstance(chunk, str) else chunk,
                            "more_body": True,
                        })
                else:
                    # Just a JSON response (error case)
                    body_content = response.body
                    await send({
                        "type": "http.response.body",
                        "body": body_content,
                        "more_body": False,
                    })
                    return
                
                # Send final empty body to close the response
                await send({
                    "type": "http.response.body",
                    "body": b"",
                    "more_body": False,
                })
                return
            except Exception as e:
                import traceback
                traceback.print_exc()
                error_response = JSONResponse(
                    {"jsonrpc": "2.0", "error": {"code": -32603, "message": str(e)}, "id": body_json.get("id")},
                    status_code=500
                )
                await error_response(scope, receive_wrapper, send)
                return
        
        # For all other requests, pass through to original app with body restored
        await self.app(scope, receive_wrapper, send)


# Create the base A2A app
_base_app = to_a2a(root_agent, port=8080)


class RunAgentRequest(BaseModel):
    """Request model for running the agent."""
    app_name: str
    user_id: str
    session_id: str
    new_message: dict[str, Any]
    streaming: bool = True
    state_delta: dict[str, Any] | None = None


def health_check(request):
    """Health check endpoint for Kagent.

    Returns:
        Health status
    """
    return JSONResponse({"status": "healthy"})


# Add health check route to base app
_base_app.router.routes.append(Route("/health", health_check))

# Wrap with our streaming middleware
app = A2AStreamingMiddleware(_base_app)


async def run_agent_sse(request: Request):
    """SSE endpoint for Kagent Web UI streaming support.
    
    This endpoint provides Server-Sent Events streaming for the Kagent Web UI,
    using proper A2A protocol event format.
    """
    try:
        # Parse the request body
        body = await request.json()
        req = RunAgentRequest(**body)
        
        # Generate IDs
        task_id = str(uuid.uuid4())
        context_id = req.session_id
        
        # Convert the new_message to ADK Content format
        new_message_content = types.Content(
            role=req.new_message.get("role", "user"),
            parts=[
                types.Part.from_text(text=part.get("text", ""))
                for part in req.new_message.get("parts", [])
                if part.get("text")
            ]
        )
        
        # Get or create session
        session = await runner.session_service.get_session(
            app_name=req.app_name,
            user_id=req.user_id,
            session_id=req.session_id
        )
        
        if not session:
            session = await runner.session_service.create_session(
                app_name=req.app_name,
                user_id=req.user_id,
                session_id=req.session_id
            )
        
        # Define the event generator for SSE with A2A format
        async def event_generator():
            try:
                # Create run config with SSE streaming mode
                run_config = RunConfig(streaming_mode=StreamingMode.SSE)
                
                # Run the agent asynchronously with streaming
                async for adk_event in runner.run_async(
                    user_id=req.user_id,
                    session_id=req.session_id,
                    new_message=new_message_content,
                    state_delta=req.state_delta,
                    run_config=run_config,
                ):
                    # Convert ADK event to A2A events
                    a2a_events = convert_adk_event_to_a2a_events(
                        adk_event,
                        task_id=task_id,
                        context_id=context_id,
                        app_name=req.app_name,
                        session_id=req.session_id,
                    )
                    
                    for a2a_event in a2a_events:
                        # Format as JSON-RPC SSE response
                        sse_response = {
                            "jsonrpc": "2.0",
                            "id": str(uuid.uuid4()),
                            "result": a2a_event.to_dict()
                        }
                        yield f"data: {json.dumps(sse_response)}\n\n"
                
                # Note: We don't send a separate TaskArtifactUpdateEvent because
                # the final TaskStatusUpdateEvent with final=True already contains
                # the complete message. Sending both would cause duplicate messages
                # in the Kagent Web UI.
                    
            except Exception as e:
                # Send error event
                error_status = TaskStatusUpdateEvent(
                    task_id=task_id,
                    context_id=context_id,
                    status=TaskStatus(
                        state="failed",
                        timestamp=datetime.now(timezone.utc).isoformat(),
                        message=Message(
                            message_id=str(uuid.uuid4()),
                            role="agent",
                            parts=[Part(root=TextPart(text=f"Error: {str(e)}"))]
                        )
                    ),
                    final=True
                )
                sse_response = {
                    "jsonrpc": "2.0",
                    "id": str(uuid.uuid4()),
                    "result": error_status.to_dict()
                }
                yield f"data: {json.dumps(sse_response)}\n\n"
        
        # Return streaming response with proper SSE content type
        return StreamingResponse(
            event_generator(),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "X-Accel-Buffering": "no",
            }
        )
        
    except Exception as e:
        return JSONResponse(
            {"error": str(e), "message": "Failed to process SSE request"},
            status_code=500
        )

# Health check route is added to _base_app before middleware wrapping

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
