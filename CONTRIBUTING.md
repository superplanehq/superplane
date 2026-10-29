# Contributing to SuperPlane

Thank you for your interest in contributing to SuperPlane! We welcome
contributions from the community.

## Before you start

We love contributions. For anything beyond a small fix, please talk with
the core maintainers first.

For features, larger bug fixes, or architectural changes:

1. Find or open an issue that describes the change.
2. Discuss the approach with us on the issue or on [Discord](https://discord.gg/KC78eCNsnw).
3. Wait until a maintainer assigns the issue to you before you start implementation.

This helps us avoid duplicate work. It also makes sure we agree on the
direction before you spend time building the change.

Documentation and typo fixes are always welcome as direct pull requests.

## Ways to Get Involved

There are many ways to contribute to SuperPlane:

- **Report bugs** - Help us identify and fix issues (see [Issue Tracking](docs/contributing/issue-tracking.md))
- **Suggest features** - Share your ideas for new functionality
- **Write code** - Fix bugs, implement features, or improve existing code
- **Improve documentation** - Help make our docs clearer and more comprehensive
- **Share feedback** - Let us know what works well and what could be better

## Development Quickstart

Getting started with SuperPlane development is fast. It only takes a couple of
minutes to set up your local development environment!

### Pre-requisites

The complete development is done inside of Docker, so you don't need any
programming languages, databases, or other dependencies installed directly on
your machine. Everything runs in containers managed by Make and Docker.

Before you begin, make sure you have the following:

- You are running MacOS or Linux
- [Make](https://www.gnu.org/software/make/)
- [Docker](https://www.docker.com/)

### Forking and Cloning the Repository

To begin working with SuperPlane, you'll need to fork and clone the repository:

1. **Fork** the repository on GitHub to have your own copy.
2. **Clone** your fork to your local machine:

```bash
git clone https://github.com/YOUR_USERNAME/superplane.git
cd superplane
```

### Setting Up the Development Environment

Once inside the cloned repository, bring up Docker, install dependencies and
migrations, then start the development server:

```sh
make dev.up       # Start containers
make dev.setup    # Install dependencies and prepare databases
make dev.server   # Start the development server
```

When `make dev.server` reports the app as healthy, open
[http://localhost:8000](http://localhost:8000).
On a fresh database, complete the initial owner account setup in your browser.

For daily development and additional setup, see:

- [Local development guide](docs/contributing/local-development.md) — Restarting, runner tools, and advanced configuration
- [Local database snapshot](docs/contributing/local-database-snapshot.md) — Restore an existing environment
- [Connecting to third-party services](docs/contributing/connecting-to-3rdparty-services-from-development.md) — External integrations and hosted OpenRouter

## Pull requests

Keep each pull request focused on one change.

If you are a new contributor, keep only one pull request open at a time. We
prefer to give one contribution proper attention rather than several competing
for review.

We may close a pull request for a substantial change that was not discussed
with us first.

See [Pull Requests](docs/contributing/pull-requests.md) for title format,
sign-off, and the review process.

## Additional Development Resources

### Overview

- **[Product Docs](https://docs.superplane.com)** - Complete product documentation and user guides
- **[Architecture Overview](docs/contributing/architecture.md)** - High-level system architecture and codebase structure

### Contributing

- **[Local Development Guide](docs/contributing/local-development.md)** - Everyday workflow, runner tools, and advanced configuration
- **[Discord](https://discord.gg/KC78eCNsnw)** - Join our Discord community for discussions, questions, and collaboration
- **[Issue Tracking](docs/contributing/issue-tracking.md)** - How to report bugs, use the SuperPlane Board, and understand issue types
- **[Pull Requests](docs/contributing/pull-requests.md)** - How to create pull-requests
- **[Commit Sign-off](docs/contributing/commit_sign-off.md)** - Information about the Developer's Certificate of Origin and signing off commits
- **[E2E Testing](docs/contributing/e2e-tests.md)** - Writing, running, and debugging end-to-end tests
- **[Running Multiple Local Instances](docs/contributing/multi-instance-dev.md)** - Run two SuperPlane repos side by side with separate ports
- **[Local database snapshot](docs/contributing/local-database-snapshot.md)** - Restore a dump to skip owner setup in a new local environment
- **[Using AI Agents](docs/contributing/ai-agents.md)** - Guide for using AI agents to help with development tasks
- **[Quality Standards](docs/contributing/quality.md)** - High-level principles for building maintainable, user-focused products

### Adding new integrations to SuperPlane

- **[Opening PRs for integrations](docs/contributing/integration-prs.md)** — Checklist for opening and submitting integration PRs (title, description, video, frontend/backend/docs, CI, DCO)
- **[Component Implementation](docs/contributing/component-implementations.md)** — Step-by-step instructions for creating new components or triggers
- **[Component Customization](docs/contributing/component-customization.md)** — Guide for customizing existing components or building behaviors
- **[Templates](docs/contributing/templates.md)** — Guide for creating and adding new workflow templates
- **[Integrations Board](https://github.com/orgs/superplanehq/projects/2/views/17)** — View all integration-related work on the SuperPlane Board
- **[Integration bounties](docs/contributing/bounties.md)** — How to claim, work on, and get paid for bounties via BountyHub; review and dispute process
- **[Connecting to Third-Party Services during Development](docs/contributing/connecting-to-3rdparty-services-from-development.md)**

## For maintainers

Assume good intent. Thank the contributor. Be clear and kind.

Help people land good work. Talk through the idea. Assign the issue. Review
one focused pull request.

If a pull request skipped that process, close it with care. Invite a next
step. Use a short, consistent response:

### Substantial PR without prior discussion

> Thanks for the contribution. For substantial changes, we ask contributors to
> discuss the change with us and get the issue assigned before starting
> implementation.
>
> I am going to close this PR for now. Please open or comment on the relevant
> issue first. If we agree on the direction, we can take it from there.

### Issue was not assigned

> Thanks for jumping in. We ask contributors to wait until an issue is assigned
> to them before starting implementation. This avoids duplicate work and
> competing PRs.
>
> I am closing this one for now. Please comment on the issue and wait until a
> maintainer assigns it to you.

### Not something we will merge

> Thanks for the contribution. This is not something we are going to merge in
> its current form, so I am going to close it rather than ask you to spend more
> time on it.
>
> If you would like to contribute another change, please start with an issue
> and coordinate with us before opening another PR.
