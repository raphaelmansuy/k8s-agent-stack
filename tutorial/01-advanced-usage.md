# Advanced Kagent CLI Usage

Once you've mastered the basics, you can use the `kagent` CLI for more advanced interactions.

## 🌊 Streaming Responses

For longer tasks, use the `--stream` flag to see the agent's response in real-time.

```bash
kagent invoke --namespace kagent --agent google-adk-byo-agent --task "Write a short poem about Kubernetes" --stream
```

## 📂 Using Task Files

If your task is complex or requires a large prompt, you can read it from a file.

```bash
echo "Analyze the current state of the cluster and suggest optimizations." > task.txt
kagent invoke --namespace kagent --agent google-adk-byo-agent --file task.txt
```

## 🔄 Session Management

Kagent maintains context through sessions. You can list and resume sessions.

```bash
# List all active sessions
kagent get session

# Resume a specific session
kagent invoke --namespace kagent --agent google-adk-byo-agent --session <session-id> --task "Tell me more about that."
```

## 🛠 Inspecting Tools

You can see what tools an agent has available.

```bash
kagent get tool
```

## 📊 Monitoring Logs

To debug agent behavior, watch the logs in real-time:

```bash
# View controller logs
make logs

# View specific agent logs
make adk-agent-logs
```
