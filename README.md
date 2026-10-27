<p align="center">
<a href="https://superplane.com/">
  <img src="./docs/images/background_image.png" alt="SuperPlane factory board with Backlog, Implement, Verify, and Done columns" width="100%">
</a>
</p>

<p align="center">
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="Build"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-F4D35E?style=flat-square&amp;labelColor=171714" alt="Apache 2.0"></a>
  <a href="https://discord.superplane.com"><img src="https://img.shields.io/discord/1409914582239023200?label=Discord&amp;style=flat-square&amp;labelColor=171714&amp;color=F4D35E" alt="Discord"></a>
  <a href="https://mcpservers.org/servers/superplanehq/superplane"><img src="https://mcpservers.org/badge.svg" alt="Listed on mcpservers.org"></a>
</p>

<p align="center">
  <a href="https://app.superplane.com/">Try SuperPlane</a> ·
  <a href="https://superplane.com/">Website</a> ·
  <a href="https://docs.superplane.com/">Docs</a> ·
  <a href="https://docs.superplane.com/self-hosting/overview/">Self-host</a> ·
  <a href="https://discord.superplane.com/">Discord</a> ·
  <a href="https://superplane.com/blog/">Blog</a>
</p>

## About

**Open source factory for one-shot engineering.**

SuperPlane coordinates coding agents, source control, CI, review, approvals, and the other tools your team already uses. A factory keeps each task, its automations, and the record of every run in one place.

One path starts from a backlog issue and ends in a pull request that a person can review. The same engine also runs the workflows you design. A workflow can start an agent, call a tool, wait for an event, or ask for approval.

- A **factory** holds work orders, automation lines, and the policies for a team.
- An **automation** is a canvas. It runs when a trigger fires, and it records what happens after success or failure.
- A **run** stores inputs, outputs, retries, and cost. A failed run can continue from a known point.
- You choose the models. Run SuperPlane in the cloud or on your own infrastructure.

## How the factory works

A work order enters the factory and moves through one or more automation lines. A line can produce a pull request, a note, a branch, or another reviewable result.

### Prepare a task before an agent starts

<p align="center">
  <img src="./docs/images/backlog-refinement-framed.png" alt="Task plan with clarity, complexity, and verifiability scores" width="100%">
</p>

SuperPlane can read an incoming issue and draft a plan. It scores clarity, complexity, and verifiability, and it marks decisions that still need a person. You review the plan, then start the agent when the task is clear.

### Check a change before a person merges

<p align="center">
  <img src="./docs/images/pull-request-verification-framed.png" alt="Factory task that moved from intake through implementation and pull request verification" width="100%">
</p>

A line can implement the task and then verify the pull request. SuperPlane watches status checks and review feedback. A failed check goes back to the agent with the failure output. Your team reviews the result and merges it.

Each stage is an automation, and you can change what that stage runs.

| Resource | What it does |
| --- | --- |
| **Factory** | Holds work orders, automation lines, and the policies for a team. |
| **Work order** | Records one task from intake through its final outcome. |
| **Line** | Defines the ordered stages that process a work order. |
| **Automation** | Runs an agent, calls a tool, waits for an event, or requires approval on a canvas. |
| **Run** | Tracks execution, inputs, outputs, retries, and cost for one step. |

Every step stays visible. SuperPlane keeps the event history, the execution state, and the artifacts. A failed run can resume without a custom script.

Read [how SuperPlane works](https://docs.superplane.com/get-started/how-superplane-works/).

## Getting started

- **SuperPlane Cloud:** [Sign up](https://app.superplane.com/), create a workspace, and connect a repository. Follow the [quickstart](https://docs.superplane.com/get-started/quickstart/).
- **Self-host:** Install SuperPlane with Docker Compose on one Linux VM, or on Kubernetes in AWS, Azure, or Google Cloud. See [self-hosting](https://docs.superplane.com/self-hosting/overview/).

## Built around your stack

SuperPlane coordinates the systems where engineering work already happens. You choose the coding agents and models. Each integration provides triggers and actions that you can compose on a canvas.

| Area | Examples |
| --- | --- |
| **AI and coding agents** | Claude, Cursor, OpenAI, OpenRouter, and Perplexity |
| **Source control and CI** | GitHub, GitLab, Bitbucket, Semaphore, CircleCI, and Harness |
| **Cloud and delivery** | AWS, Google Cloud, Azure, Cloudflare, Docker Hub, and Render |
| **Observability** | Datadog, Grafana, Honeycomb, New Relic, Prometheus, and Sentry |
| **Incidents and service management** | PagerDuty, Rootly, FireHydrant, incident.io, Jira, and ServiceNow |
| **Communication** | Slack, Discord, Microsoft Teams, Telegram, SendGrid, and SMTP |

[Explore every integration →](https://docs.superplane.com/components/)

## Contributing

We welcome focused pull requests, bug reports, and product ideas.

1. Read the [contributing guide](./CONTRIBUTING.md).
2. Review the [issue tracking process](./docs/contributing/issue-tracking.md).
3. Find an [open issue](https://github.com/superplanehq/superplane/issues) or an [open bounty](https://superplane.com/bounties/).
4. Discuss your approach with maintainers on [Discord](https://discord.superplane.com).

The development environment runs in Docker. Start with [`AGENTS.md`](./AGENTS.md) and the development section of [`CONTRIBUTING.md`](./CONTRIBUTING.md).

## Community and license

- [Discord](https://discord.superplane.com) for questions and support
- [Blog](https://superplane.com/blog/) for engineering notes and releases
- [Events](https://luma.com/superplane) for upcoming SuperPlane events
- [X](https://x.com/superplanehq) for product updates
- [Releases](https://github.com/superplanehq/superplane/releases) for release notes

SuperPlane is available under the [Apache License 2.0](./LICENSE).
