#!/bin/bash
# Start kagent UI with port-forwarding
# This script will keep running - press Ctrl+C to stop

set -e

echo "════════════════════════════════════════"
echo "  Kagent UI → http://localhost:8080"
echo "════════════════════════════════════════"
echo ""
echo "🌐 Starting port-forward..."
echo "   Keep this terminal open while using the UI"
echo "   Press Ctrl+C to stop"
echo ""

# Start port-forward (this will block)
kubectl port-forward -n kagent svc/kagent-ui 8080:8080
