# SuperPlane

SuperPlane is an **open source DevOps control plane** for defining and running
event-based workflows. It works across the tools you already use such as
Git, CI/CD, observability, incident response, infra, and notifications.

![SuperPlane screenshot](./screenshot.png)

## Project status

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="CI Status on Semaphore" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHub commit activity"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discord server" /></a>
</p>

This project is in alpha stage and moving quickly. Expect rough edges and occasional
breaking changes while we stabilize the core model and integrations.
If you try it and hit something confusing, please [open an issue](https://github.com/superplanehq/superplane/issues/new).
Early feedback is extremely valuable.

## What it does

- **Workflow orchestration**: Model multi-step operational workflows that span multiple systems.
- **Event-driven automation**: Trigger workflows from pushes, deploy events, alerts, schedules, and webhooks.
- **Control plane UI**: Design and manage DevOps processes; inspect runs, status, and history in a single place.
- **Shared operational context**: Keep workflow definitions and operational intent in one system instead of scattered scripts.

## How it works

- **Canvases**: You model a workflow as a directed graph (a “Canvas”) of steps and dependencies.
- **Components**: Each step is a reusable component (built-in or integration-backed) that performs an action (for example: call CI/CD, open an incident, post a notification, wait for a condition, require approval).
- **Events & triggers**: Incoming events (webhooks, schedules, tool events) match triggers and start executions with the event payload as input.
- **Execution + visibility**: SuperPlane executes the graph, tracks state, and exposes runs/history/debugging in the UI (and via the CLI).

### Example use cases

A few concrete things teams build with SuperPlane:

- **Policy-gated production deploy**: when CI finishes green, hold outside business hours, require on-call + product approval, then trigger the deploy.
- **Progressive delivery (10% → 30% → 60% → 100%)**: deploy in waves, wait/verify at each step, and rollback on failure with an approval gate.
- **Release train with a multi-repo ship set**: wait for tags/builds from a set of services, fan-in once all are ready, then dispatch a coordinated deploy.
- **“First 5 minutes” incident triage**: on incident created, fetch context in parallel (recent deploys + health signals), generate an evidence pack, and open an issue.

## Quick start

Run the latest demo container:

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

Then open [http://localhost:3000](http://localhost:3000) and follow the [quick startguide](https://docs.superplane.com/get-started/quickstart/).

## Supported Integrations

SuperPlane integrates with the tools you already use. Each integration provides triggers (events that start workflows) and components (actions you can run).

> View the full list in our [documentation](https://docs.superplane.com/components/). Missing a provider? [Open an issue](https://github.com/superplanehq/superplane/issues/new) to request it.

### AI & LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cursor/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cursor.svg" alt="Cursor"/><br/>Cursor</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
</tr>
</table>

### Version Control & CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/circleci/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/circleci.svg" alt="CircleCI"/><br/>CircleCI</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
</tr>
</table>

### Cloud & Infrastructure

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#ecr-•-on-image-push" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.ecr.svg" alt="AWS ECR"/><br/>AWS ECR</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#lambda-•-run-function" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.lambda.svg" alt="AWS Lambda"/><br/>AWS Lambda</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#code-artifact-•-on-package-version" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.codeartifact.svg" alt="AWS CodeArtifact"/><br/>AWS CodeArtifact</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#cloud-watch-•-on-alarm" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.cloudwatch.svg" alt="AWS CloudWatch"/><br/>AWS CloudWatch</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#sns-•-on-topic-message" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.sns.svg" alt="AWS SNS"/><br/>AWS SNS</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudflare/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudflare.svg" alt="Cloudflare"/><br/>Cloudflare</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/hetzner-cloud/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/hetzner.svg" alt="Hetzner Cloud"/><br/>Hetzner Cloud</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dockerhub/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/docker.svg" alt="DockerHub"/><br/>DockerHub</a></td>
</tr>
</table>

### Observability

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/datadog/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/datadog.svg" alt="DataDog"/><br/>DataDog</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dash0/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/dash0.svg" alt="Dash0"/><br/>Dash0</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/prometheus/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/prometheus.svg" alt="Prometheus"/><br/>Prometheus</a></td>
</tr>
</table>

### Incident Management

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/pagerduty/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/pagerduty.svg" alt="PagerDuty"/><br/>PagerDuty</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/rootly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/rootly.svg" alt="Rootly"/><br/>Rootly</a></td>
</tr>
</table>

### Communication

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/discord/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/discord.svg" alt="Discord"/><br/>Discord</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/slack/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/slack.svg" alt="Slack"/><br/>Slack</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/sendgrid/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/sendgrid.svg" alt="SendGrid"/><br/>SendGrid</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/smtp/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/smtp.svg" alt="SMTP"/><br/>SMTP</a></td>
</tr>
</table>

### Ticketing

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/jira/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/jira.svg" alt="Jira"/><br/>Jira</a></td>
</tr>
</table>

### Developer Tools

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
</tr>
</table>

## Production installation

You can deploy SuperPlane on a single host or on Kubernetes:

- **[Single Host Installation](https://docs.superplane.com/installation/overview/#single-host-installation)** - Deploy on AWS EC2, GCP Compute Engine, or other cloud providers
- **[Kubernetes Installation](https://docs.superplane.com/installation/overview/#kubernetes)** - Deploy on GKE, EKS, or any Kubernetes cluster

## Roadmap Overview

This section gives a quick snapshot of what SuperPlane already supports and what’s coming next.

**Available now**

✓ 75+ components  
✓ Event-driven workflow engine  
✓ Visual Canvas builder  
✓ Run history, event chain view, debug console  
✓ Starter CLI and example workflows

**In progress / upcoming**

→ 200+ new components (AWS, Grafana, DataDog, Azure, GitLab, Jira, and more)  
→ [Canvas version control](https://github.com/superplanehq/superplane/issues/1380)  
→ [SAML/SCIM](https://github.com/superplanehq/superplane/issues/1377) with [extended RBAC and permissions](https://github.com/superplanehq/superplane/issues/1378)  
→ [Artifact version tracking](https://github.com/superplanehq/superplane/issues/1382)  
→ [Public API](https://github.com/superplanehq/superplane/issues/1854)

## Contributing

We welcome your bug reports, ideas for improvement, and focused PRs.

- Read the **[Contributing Guide](CONTRIBUTING.md)** to get started.
- Issues: use GitHub issues for bugs and feature requests.

## License

Apache License 2.0. See `LICENSE`.

## Community

- **[Discord](https://discord.superplane.com)** - Join our community for discussions, questions, and collaboration
- **[X](https://x.com/superplanehq)** - Follow us for updates and announcements
