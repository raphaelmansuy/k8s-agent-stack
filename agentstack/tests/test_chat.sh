#!/bin/bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjYwNjQ5ODIsInByb2plY3RfaWQiOiJwcmpfZGV2Iiwic2NvcGVzIjpbImFnZW50OnJlYWQiLCJhZ2VudDppbnZva2UiLCJjaGF0OndyaXRlIl0sInRlYW1faWQiOiJ0ZWFtX2RldiIsInVzZXJfaWQiOiJ1c2VyX3Rlc3QifQ.xT_kGCxiU9VuzGt-npoHQkgZ1ULvirzyT4w_k57-5So"
API_URL="http://localhost:8081"

# Start port-forward in background
echo "Starting port-forward..."
kubectl port-forward -n agentstack svc/agentstack-api 8081:8080 > /dev/null 2>&1 &
PF_PID=$!

# Wait for port-forward to be ready
for i in {1..10}; do
    if curl -s http://localhost:8081/health > /dev/null; then
        echo "Port-forward ready"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "Port-forward failed to start"
        kill $PF_PID
        exit 1
    fi
    sleep 1
done

# Cleanup on exit
trap "kill $PF_PID" EXIT

echo "1. Creating chat session..."
RESPONSE=$(curl -s -X POST "$API_URL/v1/chat/sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "google-adk-byo-agent", "title": "Test Session"}')

echo "Response: $RESPONSE"
SESSION_ID=$(echo $RESPONSE | jq -r '.id')

if [ "$SESSION_ID" == "null" ] || [ -z "$SESSION_ID" ]; then
  echo "Failed to create session"
  exit 1
fi

echo "Session ID: $SESSION_ID"

echo "2. Sending message..."
curl -s -X POST "$API_URL/v1/chat/sessions/$SESSION_ID/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello, who are you?"}' | jq .
