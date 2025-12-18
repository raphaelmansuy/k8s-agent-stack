#!/bin/bash
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
