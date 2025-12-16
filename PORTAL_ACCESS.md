# 🎯 Kagent Portal Access - Complete Guide Index

**Last Updated**: 2025-12-16  
**Status**: ✅ **PORTAL DEPLOYED & OPERATIONAL**

---

## 🚀 Quick Start (30 seconds)

```bash
# Make sure you're in the repository
cd /Users/raphaelmansuy/Github/40-labs/cloudrun-like

# Option 1: Using Make (Recommended)
make portal-access

# Option 2: Using Script
bash access-portal.sh

# Option 3: Manual kubectl
kubectl port-forward -n kagent svc/kagent-web 3000:3000
# Then in another terminal: open http://localhost:3000
```

Then open your browser to: **`http://localhost:3000`**

You should see a dashboard with:
- ✅ System status
- ✅ Active agents list
- ✅ Quick commands
- ✅ Documentation links

---

## 📚 Documentation Structure

### 1. **THIS FILE: Portal Access Guide**
Your current file - entry point for understanding portal access

### 2. **PORTAL_QUICK_ACCESS.md**
- Fast reference guide (30-60 min use)
- Common workflows
- Troubleshooting tips
- Access methods
- **Start here for immediate use**

### 3. **PORTAL_STATUS.md**
- Current deployment status
- Health verification
- Make command reference
- Architecture diagram
- **Check here for status**

### 4. **KAGENT_PORTAL_ACCESS.md**
- Comprehensive documentation (reference)
- Installation details
- Advanced features
- Security considerations
- **Deep dive reference**

### 5. **kagent-portal.yaml**
- Kubernetes deployment manifest
- ConfigMap with HTML dashboard
- Service definitions
- Knative configuration
- **Technical implementation**

### 6. **access-portal.sh**
- Automated setup script
- One-command portal access
- Auto-deploys if needed
- Opens browser automatically
- **Convenience script**

---

## 🎯 Choose Your Document

### "I want to access the portal RIGHT NOW"
→ **Read**: [PORTAL_QUICK_ACCESS.md](PORTAL_QUICK_ACCESS.md)  
→ **Run**: `make portal-access`  
→ **Open**: http://localhost:3000

### "I want to see current status and health"
→ **Read**: [PORTAL_STATUS.md](PORTAL_STATUS.md)  
→ **Run**: `make portal-status`

### "I want comprehensive documentation"
→ **Read**: [KAGENT_PORTAL_ACCESS.md](KAGENT_PORTAL_ACCESS.md)  
→ **Topics**: Architecture, features, security, troubleshooting

### "I want to understand how it's deployed"
→ **Read**: [kagent-portal.yaml](kagent-portal.yaml)  
→ **Topics**: Kubernetes manifests, configuration, resources

### "I want one-command access"
→ **Run**: `bash access-portal.sh`  
→ **Result**: Portal opens automatically

---

## 🔄 Complete Access Workflow

### Step 1: Understand What You're Getting
```
Kagent Portal = Web UI for managing AI agents

Features:
├── Dashboard: See system status at a glance
├── Active Agents: View all deployed agents
├── Quick Commands: Copy/run common tasks
├── Documentation: Links to full guides
└── System Info: Kubernetes & component details
```

### Step 2: Choose Access Method

**Option A: Make (Recommended)**
```bash
make portal-access
# One command, works reliably
# Integrated with Make workflow
```

**Option B: Script (Automated)**
```bash
bash access-portal.sh
# Auto-deploys if needed
# Opens browser automatically
# Good for infrequent use
```

**Option C: Manual (Maximum Control)**
```bash
# Terminal 1: Start port-forward
kubectl port-forward -n kagent svc/kagent-web 3000:3000

# Terminal 2: Open browser
open http://localhost:3000

# Terminal 3: Check logs
make portal-logs
```

### Step 3: Open Browser
```
URL: http://localhost:3000
Browser: Any modern browser (Chrome, Safari, Firefox, Edge)
```

### Step 4: Explore Dashboard

**Tab 1: 📊 Dashboard**
- System online/offline status
- Kubernetes connected (yes/no)
- Agents deployed count
- Portal active status
- → If all green, everything works!

**Tab 2: 🤖 Active Agents**
- Lists: google-adk-agent
- Shows: Agent name, description
- Provides: Access URLs and ports
- Actions: Direct connection links

**Tab 3: 🛠️ Quick Commands**
- Copy command blocks
- Common tasks: port-forward, status, logs
- Paste into terminal and run
- Example: `kubectl port-forward -n kagent svc/google-adk-agent...`

**Tab 4: 📋 Make Reference**
- All Make targets listed
- Descriptions for each
- Copy functionality
- Examples: `make agent-status`, `make verify`

**Tab 5: 🔧 System Components**
- Kubernetes cluster info
- Node details
- Namespace configuration
- Ingress setup
- Network info

**Tab 6: 📖 Documentation**
- Link to Portal Quick Access Guide
- Link to Full Documentation
- Link to Getting Started
- Link to Troubleshooting

### Step 5: Use What You Find

**If you see:**
```
✓ System Online
✓ Kubernetes: Connected
✓ Agents: Deployed (1)
✓ Portal: Active
```
→ Everything works! Continue to explore agents.

**If you see:**
```
✗ Some service offline
? Network issue
```
→ See [PORTAL_QUICK_ACCESS.md#troubleshooting](PORTAL_QUICK_ACCESS.md) for fixes.

### Step 6: Stop Portal (When Done)
```bash
# In the terminal where you ran "make portal-access"
# Press: Ctrl+C

# Or stop the port-forward
pkill -f "port-forward.*kagent-web"
```

---

## 🔗 Common Tasks

### Task: Check If Portal Is Running
```bash
make portal-status
# Output: Shows pod status, service info, health
```

### Task: View Portal Logs
```bash
make portal-logs
# Output: Last 20 lines of nginx activity
```

### Task: Redeploy Portal
```bash
make portal-clean    # Remove it
make portal-deploy   # Deploy again
make portal-access   # Access it
```

### Task: Copy an Agent Access Command
1. Open http://localhost:3000
2. Click "🛠️ Quick Commands" tab
3. Find the command for your agent
4. Click to copy
5. Paste in terminal

### Task: Check System Health
1. Open http://localhost:3000
2. Look at "📊 Dashboard" tab
3. All green = healthy
4. Any red = issue (see troubleshooting)

---

## 🆘 Quick Fixes

### Problem: "Connection refused" on localhost:3000

**Fix 1:**
```bash
# Make sure port-forward is still running
make portal-access
```

**Fix 2:**
```bash
# Kill existing connections and restart
pkill -f "port-forward.*kagent-web"
make portal-access
```

**Fix 3:**
```bash
# Check portal pod is running
kubectl get pods -n kagent -l app=kagent-web
# Should show: 1/1 Running
```

### Problem: "Port 3000 already in use"

**Fix:**
```bash
# Find what's using it
lsof -i :3000

# Kill it
kill -9 <PID>

# Or use different port
kubectl port-forward -n kagent svc/kagent-web 3001:3000
# Then: http://localhost:3001
```

### Problem: "No agents showing in portal"

**Fix:**
```bash
# Check agents are deployed
kubectl get all -n kagent

# Refresh browser: Cmd+Shift+R
# Or clear cache: Cmd+Shift+Delete
```

### Problem: "Portal loads slowly"

**Fix:**
```bash
# Clear browser cache
# Close and reopen http://localhost:3000
# Or restart port-forward
make portal-access
```

---

## 📊 Portal Status Right Now

### Current Deployment
```
✅ Pod: kagent-web-6b7c9888d7-r728s
✅ Status: 1/1 Running
✅ Restarts: 0 (healthy)
✅ Service: kagent-web (ClusterIP:3000)
✅ Dashboard: Accessible at http://localhost:3000
✅ Nginx: Serving successfully
```

### What's Deployed
```
ConfigMap:     kagent-portal-config (HTML dashboard)
Deployment:    kagent-web (nginx:alpine)
Service:       kagent-web (ClusterIP on 3000)
Knative:       kagent-portal (auto-scaling ready)
```

### Agents Visible
```
google-adk-agent (Cloud Run Hello Container)
- Status: Running
- Type: Knative Service
- Access: Via port-forward
```

---

## 🛠️ Make Commands Reference

### Portal Management
```bash
make portal-deploy    # Deploy/redeploy portal
make portal-access    # Start port-forward and open browser
make portal-status    # Check health and status
make portal-logs      # Show portal activity logs
make portal-clean     # Remove portal completely
```

### Agent Management
```bash
make agent-deploy     # Deploy google-adk-agent
make agent-logs       # Show agent logs
make agent-status     # Check agent status
make port-forward     # Forward agent port
```

### System
```bash
make verify           # Full system verification
make help            # Show all available commands
```

---

## 🎯 Learning Paths

### Path 1: "I just want to see the portal" (5 minutes)
1. Run: `make portal-access`
2. Browser opens automatically
3. Look at dashboard
4. That's it! You're done.

### Path 2: "I want to understand what's here" (15 minutes)
1. Read: [PORTAL_QUICK_ACCESS.md](PORTAL_QUICK_ACCESS.md)
2. Run: `make portal-access`
3. Explore each tab in dashboard
4. Try copying a command
5. Read: [PORTAL_STATUS.md](PORTAL_STATUS.md)

### Path 3: "I want complete documentation" (30 minutes)
1. Read: [KAGENT_PORTAL_ACCESS.md](KAGENT_PORTAL_ACCESS.md) (full reference)
2. Look at: [kagent-portal.yaml](kagent-portal.yaml) (implementation)
3. Run: `make portal-access`
4. Explore portal
5. Read: Architecture and troubleshooting sections

### Path 4: "I want to troubleshoot/customize" (1+ hours)
1. Read: All documentation files
2. Study: [kagent-portal.yaml](kagent-portal.yaml)
3. Review: ConfigMap HTML/CSS/JS dashboard
4. Make changes as needed
5. Test changes
6. Redeploy: `make portal-deploy`

---

## 📱 Access from Other Devices

### From Same Local Network
```bash
# Find your machine IP
ifconfig | grep "inet " | head -1

# Use that IP instead of localhost
# On other device: http://YOUR_MACHINE_IP:3000
```

### From Remote Location
```bash
# Option 1: ngrok (recommended)
brew install ngrok
ngrok http 3000
# Share the https URL

# Option 2: SSH tunnel
ssh -L 3000:localhost:3000 your-machine
# On local device: http://localhost:3000

# Option 3: VPN
# Connect to VPN first
# Then access via internal IP
```

---

## 🔐 Security Notes

### Current Setup
- ✅ Internal only (no external access without port-forward)
- ✅ Kubernetes RBAC enforced
- ✅ No authentication needed (internal network)
- ✅ nginx running as unprivileged container

### For Production Use
- [ ] Add TLS/HTTPS certificate
- [ ] Add authentication (OAuth/JWT)
- [ ] Use Ingress with auth middleware
- [ ] Enable request logging/monitoring
- [ ] Set up backup and disaster recovery

---

## 🎓 Next Steps

### Immediate
- [ ] Run `make portal-access`
- [ ] View dashboard at http://localhost:3000
- [ ] Explore the tabs
- [ ] Note any agents or issues

### Short Term
- [ ] Bookmark the portal URL
- [ ] Create shell alias: `alias portal='make portal-access'`
- [ ] Try copying a command from portal
- [ ] Run it in terminal to test

### Medium Term
- [ ] Deploy additional agents
- [ ] Monitor portal in background
- [ ] Use portal as your main management interface
- [ ] Customize dashboard if needed

### Long Term
- [ ] Set up external access
- [ ] Add authentication
- [ ] Enable TLS/HTTPS
- [ ] Create custom agents

---

## 💡 Pro Tips

### Tip 1: Create Alias
```bash
echo "alias portal='make portal-access'" >> ~/.zshrc
source ~/.zshrc

# Then just type: portal
portal
```

### Tip 2: Permanent Port-Forward
```bash
# Put in terminal multiplexer (tmux/screen)
# Start at boot time
# Keeps portal always accessible

# Or add to crontab:
# @reboot make -C /path/to/repo portal-access
```

### Tip 3: Monitor in Background
```bash
# Terminal 1: Portal
make portal-access

# Terminal 2: Logs
watch -n 5 'make portal-logs'

# Terminal 3: Status
watch -n 10 'make portal-status'

# Terminal 4: Your work
```

### Tip 4: Link Portal in Bookmark Bar
1. Open http://localhost:3000
2. Bookmark it
3. Add to bookmarks bar
4. Click anytime to access

### Tip 5: Use Portal in Scripts
```bash
#!/bin/bash
# Auto-setup portal for deployment scripts
make portal-deploy 2>/dev/null || true
make portal-access &

# Wait for portal to be ready
sleep 5

# Your code here using portal...
```

---

## 📞 Support Matrix

| Issue | Quick Fix | Full Guide |
|-------|-----------|-----------|
| Connection refused | `make portal-access` | PORTAL_QUICK_ACCESS.md |
| Port in use | `pkill -f port-forward` | PORTAL_QUICK_ACCESS.md |
| Slow loading | Clear browser cache | PORTAL_STATUS.md |
| No agents showing | Refresh browser | PORTAL_QUICK_ACCESS.md |
| Pod not running | `make portal-deploy` | KAGENT_PORTAL_ACCESS.md |
| Want to customize | Edit ConfigMap | kagent-portal.yaml |
| Need architecture | See diagrams | KAGENT_PORTAL_ACCESS.md |

---

## ✅ Final Checklist

Before you start using the portal:

- [ ] You've read this file
- [ ] You understand your options (Make, script, manual)
- [ ] You know how to start: `make portal-access`
- [ ] You know the portal URL: http://localhost:3000
- [ ] You've looked at PORTAL_QUICK_ACCESS.md
- [ ] You know where to find help (this guide)

Before you troubleshoot:

- [ ] You've checked pod status: `make portal-status`
- [ ] You've checked logs: `make portal-logs`
- [ ] You've tried clearing browser cache
- [ ] You've tried `make portal-access` again
- [ ] You've read the troubleshooting section

---

## 🎉 YOU'RE READY!

Your Kagent Portal is deployed, running, and ready to use.

**To get started right now:**

```bash
cd /Users/raphaelmansuy/Github/40-labs/cloudrun-like
make portal-access
```

Browser opens → http://localhost:3000 → Explore dashboard → Enjoy! 🚀

---

## 📑 Quick Reference

| Need | Command/Link |
|------|---|
| Quick start | `make portal-access` |
| Fast reference | [PORTAL_QUICK_ACCESS.md](PORTAL_QUICK_ACCESS.md) |
| Status check | `make portal-status` |
| Full docs | [KAGENT_PORTAL_ACCESS.md](KAGENT_PORTAL_ACCESS.md) |
| Implementation | [kagent-portal.yaml](kagent-portal.yaml) |
| Automation | `bash access-portal.sh` |
| View logs | `make portal-logs` |
| Redeploy | `make portal-deploy` |
| Clean up | `make portal-clean` |
| Portal URL | http://localhost:3000 |

---

**Questions?** Check one of the guide documents above or run `make help` to see all available commands.

**Ready?** Type: `make portal-access` and press Enter! 🚀
