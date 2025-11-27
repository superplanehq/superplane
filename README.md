# Superplane

**Cross-platform DevOps workflow orchestration that connects your tools, automates your processes, and gives you complete visibility.**

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
- [Documentation](#documentation)
 
---

## Overview

Superplane creates a control layer above your existing DevOps tools, letting you orchestrate workflows across multiple platforms from one place. Instead of writing custom scripts and managing workflows separately in each tool, you build visual workflows that coordinate everything automatically.

---

## Key Features

- **Cross-Platform Integration** - Connect GitHub, Semaphore, CI/CD platforms, and custom webhooks in unified workflows
- **Event-Driven Automation** - Respond to pushes, deployments, alerts, and custom triggers automatically
- **Visual Workflow Builder** - See your entire DevOps process at a glance with real-time status updates
- **Enterprise Security** - Encrypted secrets, role-based access, complete audit trails
- **Centralized Monitoring** - Single dashboard for all your DevOps activities across tools
- **Flexible Deployment** - Cloud-hosted, self-hosted, or local development options

---

## Quick Start

Get your first workflow running in 10 minutes:

### Option 1: Cloud (Recommended)
1. Sign up at [app.superplane.com](https://app.superplane.com)
2. Create your first Canvas (workspace)
3. Connect a tool (GitHub, Semaphore, etc.)
4. Build a workflow using the visual editor

### Option 2: Try Locally
```bash
# Clone and start Superplane locally
git clone https://github.com/your-org/superplane
cd superplane
# Setup the environment (first time)
make dev.setup && make dev.start

# Open http://localhost:8000

# Update DB after changes

make db.migrate DB_NAME=superplane_dev

```

→ [Complete Quick Start Guide](docs/getting-started/quick-start.md)

---

## Documentation

### Getting Started
- **[What is Superplane?](docs/getting-started/what-is-superplane.md)** - Learn how Superplane solves DevOps integration challenges
- **[Quick Start Guide](docs/getting-started/quick-start.md)** - Get running in 10 minutes with 3 progressive levels
- **[Core Concepts](docs/getting-started/core-concepts.md)** - Understand Canvases, Components, and Workflows

### Installation
- **[Local Development](docs/installation/local-development.md)** - Set up and run locally
- **[Tunnels](docs/installation/tunnels.md)** - Configure tunnels for local webhook testing

### Core Concepts
- **[Canvas & Workflows](docs/concepts/canvas-and-workflows.md)** - Workspaces and workflow organization
- **[Components](docs/concepts/components.md)** - Event sources, stages, and executors
- **[Events & Data Flow](docs/concepts/events-and-data.md)** - How data moves through workflows
- **[Integrations & Security](docs/concepts/integrations-security.md)** - Connecting external tools and managing secrets

### Reference
- **[CLI Reference](docs/reference/cli.md)** - Command-line tool documentation
- **[API Documentation](https://app.superplane.com/api/v1/docs)** - REST API reference

### Examples
- **[YAML Examples](docs/examples/)** - Sample configurations for Canvas, Stages, Event Sources, and Secrets
