<div align="center">

# SuperPlane

**AI software factory for one-shot routine engineering work.**

SuperPlane turns high-confidence backlog issues into verified, review-ready
pull requests so engineers can focus on work that needs judgment.

[![Beta](https://img.shields.io/badge/status-beta-F4D35E?style=flat-square&labelColor=171714)](https://github.com/superplanehq/superplane)
[![Build](https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields)](https://superplanehq.semaphoreci.com/projects/superplane)
[![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-F4D35E?style=flat-square&labelColor=171714)](LICENSE)
[![Discord](https://img.shields.io/discord/1409914582239023200?label=Discord&style=flat-square&labelColor=171714&color=F4D35E)](https://discord.superplane.com)

[Website](https://superplane.com) ·
[Documentation](https://docs.superplane.com) ·
[GitHub organization](https://github.com/superplanehq) ·
[Discord](https://discord.superplane.com)

</div>

![SuperPlane software factory moving routine engineering work from backlog to a reviewed pull request](./docs/images/superplane-hero.png)

> [!NOTE]
> SuperPlane is in beta. We are building and refining the core primitives for
> an open source AI software factory. Follow the
> [latest releases](https://github.com/superplanehq/superplane/releases),
> explore [open issues](https://github.com/superplanehq/superplane/issues) and
> see [how you can contribute](#contributing) as the system matures.

## About

SuperPlane lets AI agents fully automate routine development work. It
coordinates coding agents, source control, CI, review, approvals and feedback
in one visible system so engineers do not have to project-manage every step.

The factory continuously evaluates which backlog issues agents can handle with
high confidence. It applies the same workflow-level guardrails to every run,
checks the result and sends failures back to the agent with actionable context.
Ambiguous work and decisions that need human judgment stay with your team.

SuperPlane is built on an Apache 2.0 open source engine. Choose your models,
run in the cloud or on-prem and build the factory around your stack and cost
constraints.

## From backlog to reviewed work

| 01 / Discover | 02 / Control | 03 / Verify |
| --- | --- | --- |
| **Investigate what can be automated.** Scan the codebase and backlog, then rank focused issues using signals such as scope, context and test coverage. | **Set the rules every agent must follow.** Define allowed scope, required checks, review policies, approval points and escalation paths once. | **Watch work finish without babysitting.** Capture failures, give the agent actionable feedback, rerun checks and return a verified pull request. |

```text
Focused issue → Plan → Build → Check → Review → Review-ready pull request
```

The workflow, not the individual agent, owns the rules. If a task still needs
human judgment, SuperPlane pauses and escalates it with the full context intact.

## How the factory works

![SuperPlane work order analysis running through a guarded automation line](./docs/images/factory-backlog.png)

A SuperPlane Factory turns incoming work into a durable operational record.
A work order enters through the backlog, moves through one or more automation
lines and produces pull requests, notes, branches and other reviewable
artifacts.

| Resource | What it does |
| --- | --- |
| **Factory** | Holds work orders, automation lines and the policies for a team. |
| **Work order** | Records one delegated task from intake through its final outcome. |
| **Line** | Defines the ordered stages that process a work order. |
| **Automation** | Runs an agent, calls a tool, waits for an event or requires approval through a Canvas-backed app. |
| **Run** | Tracks durable execution, inputs, outputs, retries and cost for one automation step. |

Every step stays visible. SuperPlane preserves the event history, execution
state and artifacts across retries so a failed run can resume without custom
glue or lost context.

## Built around your stack

SuperPlane coordinates the systems where engineering work already happens.
Each integration provides event triggers and actions that you can compose on a
Canvas.

| Area | Examples |
| --- | --- |
| **AI and coding agents** | Claude, Cursor, OpenAI, OpenRouter and Perplexity |
| **Source control and CI** | GitHub, GitLab, Bitbucket, Semaphore, CircleCI and Harness |
| **Cloud and delivery** | AWS, Google Cloud, Azure, Cloudflare, Docker Hub and Render |
| **Observability** | Datadog, Grafana, Honeycomb, New Relic, Prometheus and Sentry |
| **Incidents and service management** | PagerDuty, Rootly, FireHydrant, incident.io, Jira and ServiceNow |
| **Communication** | Slack, Discord, Microsoft Teams, Telegram, SendGrid and SMTP |

[Explore every integration →](https://docs.superplane.com/components/)

## Repository activity

Open source work should be observable. The dashboard below captures the
current project pulse in the same yellow and black visual system as SuperPlane.

<a href="https://github.com/superplanehq/superplane/pulse">
  <img src="./docs/images/repository-pulse.svg" alt="SuperPlane repository activity with recent commits, open issues, open pull requests and top contributors" width="100%">
</a>

<p align="center">
  <a href="https://github.com/superplanehq/superplane/commits/main"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane?style=for-the-badge&label=COMMITS%20THIS%20MONTH&labelColor=171714&color=F4D35E" alt="Commits this month"></a>
  <a href="https://github.com/superplanehq/superplane/issues"><img src="https://img.shields.io/github/issues/superplanehq/superplane?style=for-the-badge&label=OPEN%20ISSUES&labelColor=171714&color=F4D35E" alt="Open issues"></a>
  <a href="https://github.com/superplanehq/superplane/pulls"><img src="https://img.shields.io/github/issues-pr/superplanehq/superplane?style=for-the-badge&label=OPEN%20PRS&labelColor=171714&color=F4D35E" alt="Open pull requests"></a>
  <a href="https://github.com/superplanehq/superplane/graphs/contributors"><img src="https://img.shields.io/github/contributors/superplanehq/superplane?style=for-the-badge&label=CONTRIBUTORS&labelColor=102019&color=58D68D" alt="Contributors"></a>
</p>

Follow the [full activity feed](https://github.com/superplanehq/superplane/activity),
review [open pull requests](https://github.com/superplanehq/superplane/pulls)
or see the [latest release](https://github.com/superplanehq/superplane/releases/latest).

## Contributing

We welcome focused pull requests, bug reports and product ideas.

1. Read the [contributing guide](./CONTRIBUTING.md).
2. Review the [issue tracking process](./docs/contributing/issue-tracking.md).
3. Find an [open issue](https://github.com/superplanehq/superplane/issues) or an
   [open bounty](https://superplane.com/bounties/).
4. Join the [Discord community](https://discord.superplane.com) to discuss an
   approach with maintainers.

Contributor setup is Docker-based. Start with the repository instructions in
[`AGENTS.md`](./AGENTS.md) and the development section of
[`CONTRIBUTING.md`](./CONTRIBUTING.md).

## Community

- [Discord](https://discord.superplane.com) for discussions and support
- [Blog](https://superplane.com/blog/) for engineering notes and releases
- [Community events](https://luma.com/superplane) for maintainer sessions
- [X](https://x.com/superplanehq) for product updates

## License

SuperPlane is available under the [Apache License 2.0](./LICENSE).

<p align="center">
  <strong>Agents should not need you to hold their hand.</strong>
</p>
