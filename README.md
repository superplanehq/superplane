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
  <a href="https://discord.superplane.com/">Discord</a>
</p>

## About SuperPlane

**SuperPlane is an open source factory for one-shot engineering.** It turns
focused backlog issues into verified pull requests, so engineers can spend more
time on work that needs their judgment.

Giving an agent a task is only the beginning. Engineers still need to check its
work, investigate failed builds, and follow up on review comments. SuperPlane
coordinates that process: it assesses incoming work, runs coding agents, checks
their changes, and returns failures with context for the next attempt.

Your team defines the rules and reviews the result. SuperPlane keeps the task,
pull request, checks, and execution history together.

## How it works

### Turn backlog issues into work agents can deliver

<p align="center">
  <img src="./docs/images/backlog-refinement-framed.png" alt="Task refinement showing a plan and readiness assessment" width="100%">
</p>

Reliable automation starts with a clear task. SuperPlane analyzes the issue,
helps refine the plan, and identifies decisions that need your input before
implementation begins.

### Bring verified changes to human review

<p align="center">
  <img src="./docs/images/pull-request-verification-framed.png" alt="Pull request verification showing check fixes and merge confidence" width="100%">
</p>

SuperPlane checks the agent's work and sends failures back for correction.
An agent review assesses merge confidence and highlights remaining concerns,
so your team can make the final decision with evidence.

## Getting started

[Try SuperPlane](https://app.superplane.com/) and connect a GitHub repository.
The Factory imports issues and assesses which tasks are suitable for agent
execution. Choose a small bug, test fix, or documentation update for your first
run.

Follow the [quickstart](https://docs.superplane.com/get-started/quickstart/)
for setup, or read the [product overview](https://docs.superplane.com/get-started/overview/)
to learn how SuperPlane coordinates agents and verification.

To run SuperPlane on your infrastructure, see the
[self-hosting guide](https://docs.superplane.com/self-hosting/overview/).
It covers single-VM installations and Enterprise Kubernetes deployments.

## Choose your agents and models

Use the coding agents and models that fit your team's work and budget.
SuperPlane coordinates their work with your repository, CI checks, and pull
request feedback. Your team keeps a shared process as those choices change.

## Contributing

We welcome bug reports, documentation improvements, integrations, and code
contributions. Read the [contributing guide](./CONTRIBUTING.md)
for the contribution process and Docker-based development setup.

Before reporting a bug or requesting a feature,
[search existing issues](https://github.com/superplanehq/superplane/issues).
For substantial changes, discuss the approach with maintainers and wait for
issue assignment before implementation.

## Community and support

Join [Discord](https://discord.superplane.com/) to ask questions, share feedback,
and discuss contributions. Read the [blog](https://superplane.com/blog/) for
engineering notes, or join a [community event](https://luma.com/superplane)
to meet the team.

## License

SuperPlane is available under the
[Apache License 2.0](./LICENSE).
