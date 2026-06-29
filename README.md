# SuperPlane

SuperPlane is an open source automation engine for AI-driven engineering.

It lets you orchestrate engineering workflows across the tools you use — such as Git, LLMs, CI/CD, observability, incident tools, and infrastructure — with durable execution, approvals, and operational UI.

SuperPlane executes your processes deterministically, providing the exact guardrails both humans and AI need to safely interact with your systems.

![SuperPlane screenshot](./screenshot.png)

## Project status

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="CI Status on Semaphore" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHub commit activity"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discord server" /></a>
</p>

SuperPlane is in **beta**. Self-host the core engine ([installation guide](https://docs.superplane.com/installation/overview/)) or use [SuperPlane Cloud](https://app.superplane.com) for managed runners and one-click app installs. Core primitives and integrations are maturing; breaking changes are possible. Report issues on [GitHub](https://github.com/superplanehq/superplane/issues/new).

## What it does

SuperPlane orchestrates your existing stack into git-backed **apps** with durable execution — workflows too complex for a single script or CI job.

- **Apps**: A deployable unit combining a workflow graph, custom console UI, app-scoped memory, and deterministic execution. Versioned in git (`canvas.yaml`, `console.yaml`); defines guardrails for AI agents and human operators.
- **Event-driven orchestration**: Multi-step workflows across your Git, CI/CD, observability, incident tools, and notifications — triggered by webhooks, schedules, and tool events, with approvals, policy checks, and human-in-the-loop steps.
- **Console dashboards**: Define your own per-app operational UI as a dynamic grid of panels. Use it to display KPIs, tables, charts, runbooks, pinned nodes, and workflow controls, backed by live data from memory, runs, and executions.
- **Agents & operators**: Built-in per-app agent to design workflows and debug runs; CLI and [skills](https://github.com/superplanehq/skills) for external coding agents. Same RBAC on all paths.

## How it works

- **Canvases**: A graph of steps and their dependencies; a single canvas can express multiple workflows and run them concurrently.
- **Components**: Each node is a trigger or action, built-in or integration-backed that performs a specific task (for example: deploy a service, open an incident, post a notification, wait for a condition, require approval, etc.).
- **Events & triggers**: Incoming events match triggers and start runs with the event payload as input.
- **Runs & durable execution**: Runs, run items, and payloads are tracked across restarts; failed steps can resume without custom retry logic.
- **Memory**: App-scoped JSON storage that persists across runs.

### Example use cases

A few concrete things teams build with SuperPlane:

- **PR preview environments**: on pull request, provision an ephemeral environment, run tests, and post the live URL back to the PR.
- **Policy-gated production deploy**: when CI finishes green, hold outside business hours, require on-call + product approval, then trigger the deploy.
- **Progressive delivery (10% → 50% → 100%)**: deploy in waves, wait/verify at each step, and rollback on failure with an approval gate.
- **Release train with a multi-repo ship set**: wait for tags/builds from a set of services, fan-in once all are ready, then dispatch a coordinated deploy.
- **“First 5 minutes” incident triage**: on incident created, fetch context in parallel (recent deploys + health signals), generate an evidence pack, and open an issue.

## Quick start

**Local (demo container):**

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

Open [http://localhost:3000](http://localhost:3000).

**Cloud:** Sign up at [app.superplane.com](https://app.superplane.com) ([cloud beta overview](https://superplane.com/blog/superplane-cloud-beta/)).

For a guided first workflow, see the [quick start guide](https://docs.superplane.com/get-started/quickstart/).

## Supported Integrations

SuperPlane integrates with the tools you already use. Each integration provides triggers (events that start workflows) and components (actions you can run).

> View the full list in our [documentation](https://docs.superplane.com/components/). Missing a provider? [Open an issue](https://github.com/superplanehq/superplane/issues/new).

### AI & LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cursor/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cursor.svg" alt="Cursor"/><br/>Cursor</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/perplexity/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/perplexity.svg" alt="Perplexity"/><br/>Perplexity</a></td>
</tr>
</table>

### Version Control & CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/bitbucket/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/bitbucket.svg" alt="Bitbucket"/><br/>Bitbucket</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/circleci/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/circleci.svg" alt="CircleCI"/><br/>CircleCI</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/harness/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/harness.svg" alt="Harness"/><br/>Harness</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/octopusdeploy/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/octopus.svg" alt="Octopus Deploy"/><br/>Octopus Deploy</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
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
</tr>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/googlecloud/#cloud-build-•-create-build" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloud_build.svg" alt="GCP Cloud Build"/><br/>GCP Cloud Build</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/googlecloud/#cloud-functions-•-invoke-function" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gcp.cloudrun.svg" alt="GCP Cloud Functions"/><br/>GCP Cloud Functions</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/googlecloud/#compute-•-on-vm-instance" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gcp.svg" alt="GCP Compute"/><br/>GCP Compute</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudflare/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudflare.svg" alt="Cloudflare"/><br/>Cloudflare</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/digitalocean/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/digitalocean.svg" alt="DigitalOcean"/><br/>DigitalOcean</a></td>
</tr>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dockerhub/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/docker.svg" alt="DockerHub"/><br/>DockerHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/hetznercloud/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/hetzner.svg" alt="Hetzner Cloud"/><br/>Hetzner Cloud</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/microsoftazure/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/azure.svg" alt="Azure"/><br/>Azure</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/oraclecloudinfrastructure/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/oci.svg" alt="Oracle Cloud Infrastructure"/><br/>OCI</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/coolify/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/coolify.svg" alt="Coolify"/><br/>Coolify</a></td>
</tr>
</table>

### Observability

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/datadog/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/datadog.svg" alt="DataDog"/><br/>DataDog</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dash0/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/dash0.svg" alt="Dash0"/><br/>Dash0</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/grafana/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/grafana.svg" alt="Grafana"/><br/>Grafana</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/honeycomb/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/honeycomb.svg" alt="Honeycomb"/><br/>Honeycomb</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/logfire/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/logfire.svg" alt="Logfire"/><br/>Logfire</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/newrelic/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/newrelic.svg" alt="New Relic"/><br/>New Relic</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/prometheus/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/prometheus.svg" alt="Prometheus"/><br/>Prometheus</a></td>
</tr>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/elastic/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/elastic.svg" alt="Elastic"/><br/>Elastic</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/sentry/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/sentry.svg" alt="Sentry"/><br/>Sentry</a></td>
</tr>
</table>

### Incident Management

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/firehydrant/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/firehydrant.svg" alt="FireHydrant"/><br/>FireHydrant</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/incident/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/incident.svg" alt="Incident.io"/><br/>Incident.io</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/pagerduty/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/pagerduty.svg" alt="PagerDuty"/><br/>PagerDuty</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/rootly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/rootly.svg" alt="Rootly"/><br/>Rootly</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/statuspage/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/statuspage.svg" alt="Statuspage"/><br/>Statuspage</a></td>
</tr>
</table>

### Communication

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/discord/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/discord.svg" alt="Discord"/><br/>Discord</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/sendgrid/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/sendgrid.svg" alt="SendGrid"/><br/>SendGrid</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/slack/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/slack.svg" alt="Slack"/><br/>Slack</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/smtp/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/smtp.svg" alt="SMTP"/><br/>SMTP</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/microsoftteams/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/teams.svg" alt="Microsoft Teams"/><br/>Microsoft Teams</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/telegram/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/telegram.svg" alt="Telegram"/><br/>Telegram</a></td>
</tr>
</table>

### Ticketing

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/jira/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/jira.svg" alt="Jira"/><br/>Jira</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/servicenow/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/servicenow.svg" alt="ServiceNow"/><br/>ServiceNow</a></td>
</tr>
</table>

### Developer Tools

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudsmith/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudsmith.svg" alt="Cloudsmith"/><br/>Cloudsmith</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/jfrogartifactory/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/jfrog-artifactory.svg" alt="JFrog Artifactory"/><br/>JFrog Artifactory</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/launchdarkly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/launchdarkly.svg" alt="LaunchDarkly"/><br/>LaunchDarkly</a></td>
</tr>
</table>

## Security

- **RBAC** and **service accounts** for API access — see [access control](https://docs.superplane.com/concepts/access-control/) and [service accounts](https://docs.superplane.com/concepts/service-accounts/)
- **Secrets** stored encrypted — see [secrets](https://docs.superplane.com/concepts/secrets/)

## Production installation

**Footprint:** PostgreSQL, RabbitMQ, and the SuperPlane application (bundled in the single-host Docker Compose installer). Deploy on a single Linux host or Kubernetes (GKE, EKS) with external PostgreSQL. See the [installation overview](https://docs.superplane.com/installation/overview/) for topology-specific upgrade paths.

- **[Single Host Installation](https://docs.superplane.com/installation/overview/#single-host-installation)** — AWS EC2, GCP Compute Engine, Hetzner, DigitalOcean, Linode, or any Linux server
- **[Kubernetes Installation](https://docs.superplane.com/installation/overview/#kubernetes)** — GKE, EKS, or any Kubernetes cluster

Installation admins can enable private network access during owner setup or later in `/admin/settings` when SuperPlane needs to reach tools inside a VPC, private Kubernetes cluster, or another closed network. Environment variables still take precedence: set `BLOCKED_HTTP_HOSTS` or `BLOCKED_PRIVATE_IP_RANGES` to override the UI-controlled policy, and set either variable to an empty value to disable that specific block list entirely.

## Contributing

We welcome your bug reports, ideas for improvement, and focused PRs.

- Read the **[Contributing Guide](CONTRIBUTING.md)** to get started.
- Issues: use GitHub issues for bugs and feature requests.

## License

Apache License 2.0. See `LICENSE`.

## Community

- **[Discord](https://discord.superplane.com)** - Join our community for discussions, questions, and collaboration
- **[X](https://x.com/superplanehq)** - Follow us for updates and announcements

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test

test
