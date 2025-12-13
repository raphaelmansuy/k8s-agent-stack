# Copyright 2025 Google LLC
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

import google.auth
from google.adk.a2a.utils.agent_to_a2a import to_a2a
from google.cloud import logging as google_cloud_logging

from app.agent import root_agent
from app.app_utils.telemetry import setup_telemetry

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

# Use to_a2a() to properly expose the agent via A2A protocol
# This creates a Starlette app with proper A2A endpoints including /.well-known/agent-card.json
app = to_a2a(root_agent, port=8080)

# Add health check endpoint for Kagent readiness probes using Starlette routing
from starlette.responses import JSONResponse
from starlette.routing import Route

def health_check(request):
    """Health check endpoint for Kagent.

    Returns:
        Health status
    """
    return JSONResponse({"status": "healthy"})

# Add health route to the Starlette app
app.routes.append(Route("/health", health_check, methods=["GET"]))


# Main execution
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
