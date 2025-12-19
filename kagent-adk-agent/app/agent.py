# ruff: noqa
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

import datetime
from zoneinfo import ZoneInfo

from google.adk.agents import Agent
from google.adk.apps.app import App
from google.adk.models.lite_llm import LiteLlm

import os
import google.auth

# Try to get GCP project from environment or auth, with fallback
try:
    _, project_id = google.auth.default()
    os.environ["GOOGLE_CLOUD_PROJECT"] = project_id
except google.auth.exceptions.DefaultCredentialsError:
    # If no credentials, use environment variable or default
    project_id = os.environ.get("GOOGLE_CLOUD_PROJECT", "demo-project")
    os.environ["GOOGLE_CLOUD_PROJECT"] = project_id

os.environ["GOOGLE_CLOUD_LOCATION"] = os.environ.get("GOOGLE_CLOUD_LOCATION", "us-central1")
os.environ["GOOGLE_GENAI_USE_VERTEXAI"] = os.environ.get("GOOGLE_GENAI_USE_VERTEXAI", "False")


def get_weather(query: str) -> str:
    """Simulates a web search. Use it get information on weather.

    Args:
        query: A string containing the location to get weather information for.

    Returns:
        A string with the simulated weather information for the queried location.
    """
    if "sf" in query.lower() or "san francisco" in query.lower():
        return "It's 60 degrees and foggy."
    return "It's 90 degrees and sunny."


def get_current_time(query: str) -> str:
    """Gets the current time for a city or timezone.

    Args:
        query: The name of the city or location to get the current time for.

    Returns:
        A string with the current time information.
    """
    if "sf" in query.lower() or "san francisco" in query.lower():
        tz_identifier = "America/Los_Angeles"
    elif "ny" in query.lower() or "new york" in query.lower():
        tz_identifier = "America/New_York"
    elif "london" in query.lower():
        tz_identifier = "Europe/London"
    elif "tokyo" in query.lower():
        tz_identifier = "Asia/Tokyo"
    elif "utc" in query.lower():
        tz_identifier = "UTC"
    else:
        return f"Sorry, I don't have timezone information for query: {query}."

    tz = ZoneInfo(tz_identifier)
    now = datetime.datetime.now(tz)
    return f"The current time for {query} is {now.strftime('%Y-%m-%d %H:%M:%S %Z%z')}"


def calculate_math(expression: str) -> str:
    """Evaluates a simple mathematical expression.

    Args:
        expression: A mathematical expression string (e.g., "2 + 2", "10 * 5").

    Returns:
        The result of the calculation or an error message.
    """
    try:
        # Safe evaluation of simple math expressions
        result = eval(expression, {"__builtins__": {}}, {})
        return f"The result of {expression} is {result}"
    except Exception as e:
        return f"Error calculating {expression}: {str(e)}"


# Use OpenAI via litellm for better compatibility
root_agent = Agent(
    name="root_agent",
    model=LiteLlm(model="openai/gpt-4o-mini"),  # OpenAI model via LiteLLM
    instruction="""You are a helpful AI assistant designed to provide accurate and useful information.
    
You can:
- Get weather information for locations
- Tell the current time in various cities (San Francisco, New York, London, Tokyo, UTC)
- Perform mathematical calculations
- Answer general questions using your knowledge

Always be friendly, clear, and concise in your responses.""",
    tools=[get_weather, get_current_time, calculate_math],
)

app = App(root_agent=root_agent, name="app")
