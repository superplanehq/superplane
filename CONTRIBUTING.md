# Contributing to SuperPlane

Thank you for your interest in contributing to SuperPlane! We welcome
contributions from the community.

## Ways to Get Involved

Before starting your contribution, especially for core features, we encourage
you to reach out to us on [Discord](https://discord.gg/KC78eCNsnw). This allows
us to ensure that your proposed feature aligns with the project's roadmap and
goals. Developers are the key to making SuperPlane the best platform it can be,
and we value input from the community.

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

### Setting Up the Development Environment

Once inside the cloned repository, set up your local environment and start
the app with:

```sh
make dev.setup     # Install dependencies, create the database, etc.
make dev.start     # Start the development server (UI at http://localhost:8000)
```

These commands will spin up all required services in Docker containers.

When the process completes, you can access the SuperPlane at [http://localhost:8000](http://localhost:8000).

## Additional Development Resources

### Overview

- **[Product Docs](https://docs.superplane.com)** - Complete product documentation and user guides
- **[Architecture Overview](docs/contributing/architecture.md)** - High-level system architecture and codebase structure

### Contributing

- **[Discord](https://discord.gg/KC78eCNsnw)** - Join our Discord community for discussions, questions, and collaboration
- **[Issue Tracking](docs/contributing/issue-tracking.md)** - How to report bugs, use the SuperPlane Board, and understand issue types
- **[Pull Requests](docs/contributing/pull-requests.md)** - How to create pull-requests
- **[Commit Sign-off](docs/contributing/commit_sign-off.md)** - Information about the Developer's Certificate of Origin and signing off commits
- **[E2E Testing](docs/contributing/e2e-tests.md)** - Writing, running, and debugging end-to-end tests

### Adding new integrations to SuperPlane

- **[Integrations](docs/contributing/applications.md)** — Instructions for adding new third-party integrations to SuperPlane
- **[Component Implementation](docs/development/component-implementations.md)** — Step-by-step instructions for creating new components or triggers
- **[Component Customization](docs/development/component-customization.md)** — Guide for customizing existing components or building behaviors
- **[Integrations Board](https://github.com/orgs/superplanehq/projects/2/views/17)** — View all integration-related work on the SuperPlane Board
- **[Connecting to Third-Party Services during Development](docs/contributing/connecting-to-3rdparty-services-from-development.md)**
