#!/bin/bash
#
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
#
kubectl port-forward svc/agentstack-api 8888:8080 -n agentstack > /tmp/pf.log 2>&1 &
PF_PID=$!
sleep 5

echo "Creating session..."
SESSION_RESP=$(curl -s -X POST http://127.0.0.1:8888/v1/chat/sessions \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "google-adk-byo-agent", "title": "Test Session"}')

echo "Session Response: $SESSION_RESP"
SESSION_ID=$(echo $SESSION_RESP | jq -r '.id')

if [ "$SESSION_ID" == "null" ] || [ -z "$SESSION_ID" ]; then
  echo "Failed to create session"
  kill $PF_PID
  exit 1
fi

echo "Session ID: $SESSION_ID"

echo "Sending message: 'What is 2+2?'"
MESSAGE_RESP=$(curl -s -X POST http://127.0.0.1:8888/v1/chat/sessions/$SESSION_ID/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "What is 2+2?"}')

echo "Message Response: $MESSAGE_RESP"

echo "Listing messages..."
LIST_RESP=$(curl -s http://127.0.0.1:8888/v1/chat/sessions/$SESSION_ID/messages)
echo "List Response: $LIST_RESP"

kill $PF_PID
