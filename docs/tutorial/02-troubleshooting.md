# Troubleshooting Kagent

Common issues and how to resolve them.

## ❌ 404 Errors when using `kagent invoke`

If you see a `404 page not found` error when trying to invoke an agent, it usually means the CLI cannot find the agent or the controller.

**Solution:**
- Ensure you specify the namespace: `kagent invoke --namespace kagent --agent <agent-name>`
- Check if the port-forward to the controller is running: `kubectl port-forward -n kagent svc/kagent-controller 8083:8083`

## ❌ Agent Pod is not starting

If `make adk-agent-status` shows the pod is in `Pending` or `CrashLoopBackOff`.

**Solution:**
- Check the logs: `make adk-agent-logs`
- Ensure your `OPENAI_API_KEY` is correctly set.
- Check for resource constraints in your cluster.

## ❌ UI is not accessible

If [http://localhost:8080](http://localhost:8080) doesn't load.

**Solution:**
- Ensure `make ui` is running in a separate terminal.
- Check if the `kagent-ui` service exists: `kubectl get svc -n kagent kagent-ui`

## ❌ "Pty Host disconnected" in VS Code

This can happen when pasting large blocks of code or commands into the terminal.

**Solution:**
- Run commands from the `Makefile` instead of pasting long strings.
- Use `make start`, `make adk-agent`, etc.
