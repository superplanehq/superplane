# Contributing to SuperPlane

Thank you for your interest in contributing to SuperPlane! We welcome
contributions from the community.

Before starting your contribution, especially for core features, we encourage
you to reach out to us on Discord. This allows us to ensure that your proposed
feature aligns with the project's roadmap and goals. Developers are the key to
making SuperPlane the best platform it can be, and we value input from the
community.

There are many ways to contribute to SuperPlane:

- Report bugs - Help us identify and fix issues
- Suggest features - Share your ideas for new functionality
- Write code - Fix bugs, implement features, or improve existing code
- Improve documentation - Help make our docs clearer and more comprehensive
- Share feedback - Let us know what works well and what could be better

## Development Quickstart

Getting started with SuperPlane development is fast. It only takes a couple of
minutes to set up your local development environment!

### Pre-requisites

Before you begin, make sure you have the following:

- You are running MacOS or Linux
- [Make](https://www.gnu.org/software/make/)
- [Docker](https://www.docker.com/)

The complete development is done inside of Docker, so you don't need any
programming languages, databases, or other dependencies installed directly on
your machine. Everything runs in containers managed by Make and Docker.

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

When the process completes, you can access the SuperPlane UI at [http://localhost:8000](http://localhost:8000).

## Additional Development Resources

### Contributing

- **[How to Report Bugs](docs/contributing/how-to-report-bugs.md)** - Guidelines for reporting issues effectively
- **[Development Workflow](docs/contributing/development-workflow.md)** - Day-to-day workflow for making contributions
- **[Pull Requests](docs/contributing/pull-requests.md)** - Submitting and reviewing pull requests

### Building Components

- **[Applications](docs/development/applications.md)** - Building applications on top of Superplane
- **[Component Implementations](docs/development/component-implementations.md)** - Patterns and best practices for implementing components
- **[Component Customization](docs/development/component-customization.md)** - Customizing existing components
- **[E2E Tests](docs/development/e2e_tests.md)** - Writing and running end-to-end tests
