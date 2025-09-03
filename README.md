# Superplane

**Cross-platform DevOps workflow orchestration that connects your tools, automates your processes, and gives you complete visibility.**

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
1. Sign up at [app.supperplane.com](https://app.superplane.com/app)
2. Create your first Canvas (workspace)
3. Connect a tool (GitHub, Semaphore, etc.)
4. Build a workflow using the visual editor

### Option 2: Try Locally
```bash
# Clone and start Superplane locally
git clone https://github.com/superplanehq/superplane
cd superplane
make dev.setup && make dev.start

# Open http://localhost:8080
```

→ [Complete Quick Start Guide](docs/getting-started/quick-start.md)

---

## Documentation

### Getting Started
- **[What is Superplane?](docs/getting-started/overview.md)** - Learn how Superplane solves DevOps integration challenges
- **[Quick Start Guide](docs/getting-started/quick-start.md)** - Get running in 10 minutes
- **[Core Concepts](docs/getting-started/core-concepts.md)** - Understand Canvases, Components, and Workflows

### User Guides  
- **[Your First Workflow](docs/guides/your-first-workflow.md)** - Step-by-step tutorial
- **[Setting Up Integrations](docs/guides/integrations.md)** - Connect your DevOps tools
- **[Advanced Workflows](docs/guides/advanced-workflows.md)** - Multi-step automation patterns
- **[Troubleshooting](docs/guides/troubleshooting.md)** - Common issues and solutions

### Core Concepts
- **[Canvas & Workflows](docs/concepts/canvas-and-workflows.md)** - Workspaces and workflow organization
- **[Components](docs/concepts/components.md)** - Event sources, stages, and executors
- **[Events & Data Flow](docs/concepts/events-and-data.md)** - How data moves through workflows
- **[Integrations](docs/concepts/integrations.md)** - Connecting external tools
- **[Security](docs/concepts/security.md)** - Secrets, permissions, and audit logs

### Reference
- **[CLI Reference](docs/reference/cli.md)** - Command-line tool documentation
- **[YAML Schemas](docs/reference/yaml-schemas.md)** - Complete configuration reference  
- **[API Documentation](docs/reference/api.md)** - REST API reference
- **[Integration Guides](docs/reference/integrations/)** - Tool-specific setup guides

### Examples
- **[Simple CI/CD Pipeline](docs/examples/simple-ci-cd.md)** - Basic build-test-deploy workflow
- **[Multi-Environment Deployment](docs/examples/multi-env-deployment.md)** - Staging → Production with approvals
- **[Incident Response](docs/examples/incident-response.md)** - Automated alert handling