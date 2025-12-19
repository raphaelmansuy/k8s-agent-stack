#!/bin/bash
set -e

# Configuration
ENDPOINT="http://localhost:8080"
API_KEY="test-api-key"
RANDOM_ID=$(date +%s)
PROJECT_NAME="E2E Test Project $RANDOM_ID"
AGENT_NAME="E2E Test Agent $RANDOM_ID"
DEPLOYMENT_NAME="e2e-deployment-$RANDOM_ID"

echo "Starting E2E Test..."

# 1. Create Project
echo "Creating project..."
PROJECT_ID=$(./bin/agentctl project create --name "$PROJECT_NAME" --endpoint "$ENDPOINT" --api-key "$API_KEY" -o json | jq -r '.id')
echo "Project created: $PROJECT_ID"

# 2. Create Agent
echo "Creating agent..."
AGENT_ID=$(./bin/agentctl agent create --name "$AGENT_NAME" --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY" -o json | jq -r '.id')
echo "Agent created: $AGENT_ID"

# 3. Deploy Agent
echo "Deploying agent..."
cat <<EOF > e2e-deploy.yaml
apiVersion: kagent.dev/v1alpha2
kind: Deployment
metadata:
  name: $DEPLOYMENT_NAME
spec:
  agentName: $AGENT_NAME
  image: nginx
EOF

./bin/agentctl apply -f e2e-deploy.yaml --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY"
echo "Deployment created"

# 4. Check Status
echo "Checking status..."
STATUS=$(./bin/agentctl deploy list --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY" -o json | jq -r '.[0].status.phase')
echo "Status: $STATUS"

# 5. Scale Deployment
echo "Scaling deployment..."
./bin/agentctl deploy scale "$DEPLOYMENT_NAME" --replicas 2 --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY"
REPLICAS=$(./bin/agentctl deploy list --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY" -o json | jq -r '.[0].replicas')
echo "Replicas: $REPLICAS"

# 6. Create API Key
echo "Creating API key..."
NEW_KEY=$(./bin/agentctl keys create --name "E2E Key" --endpoint "$ENDPOINT" --api-key "$API_KEY" -o json | jq -r '.api_key')
echo "New API key created"

# 7. Test Chat with new key
echo "Testing chat..."
SESSION_ID=$(curl -s -X POST "$ENDPOINT/api/v1/chat/sessions" \
  -H "Authorization: ApiKey $NEW_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"agent_id\": \"$AGENT_ID\"}" | jq -r '.id')

RESPONSE=$(curl -s -X POST "$ENDPOINT/api/v1/chat/sessions/$SESSION_ID/messages" \
  -H "Authorization: ApiKey $NEW_KEY" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello!"}' | jq -r '.assistant_message.content')

echo "Agent response: $RESPONSE"

# 8. Cleanup
echo "Cleaning up..."
./bin/agentctl deploy delete "$DEPLOYMENT_NAME" --force --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY"
./bin/agentctl agent delete "$AGENT_ID" --force --project "$PROJECT_ID" --endpoint "$ENDPOINT" --api-key "$API_KEY"
./bin/agentctl project delete "$PROJECT_ID" --force --endpoint "$ENDPOINT" --api-key "$API_KEY"

echo "E2E Test Passed!"
rm e2e-deploy.yaml
