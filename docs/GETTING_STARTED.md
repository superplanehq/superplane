# Getting Started with SuperPlane Development

Welcome! This guide will help you set up SuperPlane for local development.

## Prerequisites

Before you start, make sure you have:

- **Operating System**: macOS or Linux (Windows users can use WSL 2)
- **[Make](https://www.gnu.org/software/make/)** - Build automation tool
- **[Docker](https://www.docker.com/)** - Container platform
- **Git** - Version control

> **Why Docker?** Everything runs in containers, so you don't need Go, Node.js, PostgreSQL, or any other dependencies installed directly on your machine.

## Step-by-Step Setup

### 1. Fork and Clone the Repository

Start by creating your own copy and cloning it:

```bash
# Fork on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/superplane.git
cd superplane
```

### 2. Set Up Your Development Environment

Run the setup command to prepare everything:

```bash
make dev.setup
```

This command will:
- Build Docker images for the backend and frontend
- Create the development database
- Run database migrations
- Install dependencies

**Time to completion**: 5-10 minutes depending on your system and internet speed.

### 3. Start the Development Server

Once setup is complete, start the development environment:

```bash
make dev.start
```

This will:
- Start the backend API server (gRPC + REST)
- Start the frontend development server
- Start PostgreSQL, RabbitMQ, and other services
- Make the UI accessible at http://localhost:8000

### 4. Access SuperPlane

Open your browser and navigate to:

```
http://localhost:8000
```

You should see the SuperPlane login page. Use the default credentials or sign up as a new user.

## Common Development Commands

### Running Tests

```bash
# Run all backend tests
make test

# Run tests for a specific package
make test PKG_TEST_PACKAGES=./pkg/workers

# Run E2E tests
make e2e

# Run E2E tests for a specific package
make e2e E2E_TEST_PACKAGES=./test/e2e/workflows

# Test the current file
make test.file FILE=path/to/file_test.go

# Test at specific line
make test.line FILE=path/to/file_test.go LINE=123
```

### Building and Formatting

```bash
# Format Go code
make format.go

# Format JavaScript/TypeScript code
make format.js

# Check Go formatting
make check.format

# Lint code
make lint

# Build the app
make check.build.app

# Build the UI
make check.build.ui
```

### Database Operations

```bash
# Create a new migration (always use dashes in names, never underscores)
make db.migration.create NAME=add-user-table

# Run migrations on development database
make db.migrate DB_NAME=superplane_dev

# Run migrations on test database
make db.migrate DB_NAME=superplane_test

# View database schema
make db.structure
```

### Code Generation

After updating proto files or OpenAPI specs:

```bash
# Regenerate protobuf files
make pb.gen

# Generate OpenAPI spec
make openapi.spec.gen

# Generate Go SDK
make openapi.client.gen

# Generate TypeScript SDK
make openapi.web.client.gen
```

### Stopping Development

```bash
# Stop all containers
make dev.stop

# Clean up and reset (careful - this deletes your database!)
make dev.clean
```

## Project Structure Overview

```
superplane/
├── cmd/              # Application entry points
│   ├── cli/         # CLI tool
│   └── server/      # Server application
├── pkg/             # Core Go packages
│   ├── grpc/        # gRPC API implementation
│   ├── workers/     # Background workers
│   ├── models/      # Database models
│   ├── integrations/ # Third-party integrations
│   └── ...
├── web_src/         # Frontend (React + TypeScript)
│   ├── src/
│   │   ├── pages/   # Page components
│   │   ├── components/ # Reusable components
│   │   ├── hooks/   # React hooks
│   │   ├── lib/     # Utility functions
│   │   └── assets/  # Images, styles
│   └── ...
├── db/              # Database
│   ├── migrations/  # SQL migration files
│   └── structure.sql # Schema documentation
├── protos/          # Protocol buffer definitions
├── docs/            # Documentation
├── test/            # E2E tests
├── Makefile         # Build commands
└── docker-compose.dev.yml # Docker services
```

For a detailed explanation of the project structure, see [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md).

## Development Workflow

### Making Changes

1. **Create a branch** for your work:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes** to the relevant files

3. **Format your code**:
   ```bash
   make format.go
   make format.js
   ```

4. **Run tests** to ensure nothing broke:
   ```bash
   make test
   ```

5. **Check for linting issues**:
   ```bash
   make lint
   ```

### Committing Changes

- Use [Conventional Commits](https://www.conventionalcommits.org/) format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `chore:` for maintenance tasks
  - `docs:` for documentation changes

- Example: `git commit -m "feat: add webhook retry logic"`

- Sign off your commits (see [Commit Sign-off](contributing/commit_sign-off.md)):
  ```bash
  git commit -m "feat: add webhook retry logic" --signoff
  ```

### Opening a Pull Request

1. Push your branch to your fork
2. Create a PR with a clear title following Conventional Commits
3. Fill out the PR description with context
4. Wait for CI checks and reviews

See [Pull Requests Guide](contributing/pull-requests.md) for detailed guidelines.

## Troubleshooting

### Port Already in Use

If port 8000 or another port is already in use, you can:

```bash
# Kill the process using the port (macOS/Linux)
lsof -ti:8000 | xargs kill -9

# Or stop all containers
make dev.stop
```

### Database Connection Issues

```bash
# Reset the database and migrations
make db.reset DB_NAME=superplane_dev

# Or do a full clean slate
make dev.clean
make dev.setup
```

### Docker Issues

```bash
# Rebuild containers from scratch
docker-compose -f docker-compose.dev.yml down
docker system prune -a
make dev.setup
```

### Container Not Running?

Check logs:

```bash
docker-compose -f docker-compose.dev.yml logs -f
```

## Next Steps

- **Read the Architecture**: Understand how SuperPlane works ([Architecture Overview](contributing/architecture.md))
- **Explore the Code**: Check out the [Project Structure](PROJECT_STRUCTURE.md)
- **Pick an Issue**: Start with [good first issues](https://github.com/superplanehq/superplane/labels/good%20first%20issue)
- **Join Discord**: Chat with maintainers and other contributors ([Discord](https://discord.gg/KC78eCNsnw))

## Questions or Issues?

- Check [AGENTS.md](../AGENTS.md) for coding guidelines
- Browse existing [GitHub Issues](https://github.com/superplanehq/superplane/issues)
- Join the [Discord community](https://discord.gg/KC78eCNsnw)
- See [Contributing Guide](CONTRIBUTING.md) for more info

Happy coding! 🚀
