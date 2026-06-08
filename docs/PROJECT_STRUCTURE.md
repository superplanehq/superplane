# SuperPlane Project Structure

This document explains the organization of the SuperPlane codebase and helps you navigate the repository.

## Directory Overview

```
superplane/
├── cmd/                          # Application entry points
├── pkg/                          # Core Go packages (reusable libraries)
├── web_src/                      # Frontend application (React + TypeScript)
├── protos/                       # Protocol buffer definitions
├── db/                           # Database structure and migrations
├── docs/                         # Documentation
├── test/                         # End-to-end tests
├── scripts/                      # Build and utility scripts
├── templates/                    # Workflow templates and email templates
├── release/                      # Release management tools
├── Makefile                      # Build automation
├── docker-compose.dev.yml        # Development Docker services
└── README.md                     # Project overview
```

## Backend (Go) Structure

### `cmd/` - Application Entry Points

Entry points for the application:

```
cmd/
├── server/              # Main API server
│   ├── main.go         # Server initialization and startup
│   ├── handlers.go     # HTTP request handlers
│   └── config.go       # Configuration loading
└── cli/                # Command-line interface tool
    ├── main.go
    └── commands/       # CLI commands
```

**Purpose**: These are the executables. The server handles API requests; the CLI is a tool for users/automation.

### `pkg/` - Core Packages

Reusable Go libraries organized by functionality:

```
pkg/
├── models/                  # Database models and data structures
│   ├── user.go
│   ├── organization.go
│   ├── canvas.go
│   └── ...
├── grpc/                    # gRPC API implementation
│   ├── actions/            # Action handlers
│   ├── users/              # User service
│   ├── canvases/           # Canvas service
│   └── ...
├── workers/                # Background job processors
│   ├── workflow_event_router.go  # Routes events through workflows
│   ├── component_executor.go     # Executes components
│   ├── webhook_handler.go        # Processes webhooks
│   └── ...
├── integrations/           # Third-party integrations
│   ├── slack/
│   ├── github/
│   ├── datadog/
│   └── ...
├── components/             # Built-in components
│   ├── http_request.go
│   ├── delay.go
│   ├── condition.go
│   └── ...
├── triggers/               # Event triggers
│   ├── webhook.go
│   ├── schedule.go
│   └── ...
├── database/               # Database access and GORM setup
├── authentication/         # User authentication
├── authorization/          # Permission checks (RBAC)
├── services/              # Business logic services
├── config/                # Configuration management
├── logging/               # Structured logging
├── crypto/                # Encryption utilities
├── jwt/                   # JWT token handling
├── secrets/               # Secret management
├── registry/              # Component registry
├── core/                  # Core utilities and helpers
└── ...
```

**Key Files**:
- `models/` - Database schema definitions (what data structures look like)
- `grpc/` - REST/gRPC API endpoints (external interface)
- `workers/` - Background async processing (workflow execution engine)
- `integrations/` - Third-party APIs (Slack, GitHub, DataDog, etc.)

## Frontend Structure

### `web_src/` - React Application

```
web_src/
├── src/
│   ├── pages/                      # Page-level components
│   │   ├── canvases/              # Canvas (workflow) pages
│   │   ├── workflowv2/            # Workflow v2 UI
│   │   │   ├── mappers/           # Component mappers for each integration
│   │   │   │   ├── slack/
│   │   │   │   ├── github/
│   │   │   │   └── ...
│   │   │   └── components/        # UI components for workflow builder
│   │   ├── runs/                  # Execution history pages
│   │   ├── dashboard/             # Dashboard pages
│   │   └── ...
│   ├── components/                # Reusable React components
│   │   ├── ui/                    # Basic UI elements (Button, Modal, etc.)
│   │   ├── common/                # Shared components
│   │   └── integrations/          # Integration-specific UI components
│   ├── hooks/                     # Custom React hooks
│   │   ├── useAuth.ts
│   │   ├── useCanvas.ts
│   │   └── ...
│   ├── lib/                       # Non-React utilities
│   │   ├── api/                  # API client functions
│   │   ├── utils/                # Helper functions
│   │   └── types/                # TypeScript type definitions
│   ├── assets/                    # Static assets
│   │   ├── icons/                # SVG icons for integrations
│   │   ├── styles/               # Global CSS/styling
│   │   └── ...
│   ├── App.tsx                    # Root component
│   └── main.tsx                   # Entry point
├── index.html                     # HTML template
├── vite.config.ts                 # Vite build configuration
├── tsconfig.json                  # TypeScript configuration
├── package.json                   # Dependencies
└── AGENTS.md                      # Frontend development guidelines
```

**Key Directories**:
- `pages/` - Route-based components (what users see)
- `components/` - Reusable UI building blocks
- `hooks/` - React logic you can reuse
- `lib/` - Utilities that aren't React-specific
- `workflowv2/mappers/` - Integration-specific canvas configurations

## Database

### `db/` - Schema and Migrations

```
db/
├── structure.sql           # Complete schema documentation
├── migrations/             # SQL migration files
│   ├── 000001_init.up.sql
│   ├── 000001_init.down.sql
│   ├── 000002_add_users.up.sql
│   ├── 000002_add_users.down.sql
│   └── ...
└── data_migrations/        # Data transformation scripts
```

**Important**: 
- Migrations are applied in order
- `.up.sql` - What to do when migrating forward
- `.down.sql` - Leave empty (we don't support rollbacks)
- Use `make db.migration.create NAME=feature-name` to create new migrations

### `protos/` - API Contracts

```
protos/
├── actions.proto           # Workflow action definitions
├── canvases.proto          # Canvas (workflow) definitions
├── components.proto        # Component specifications
├── integrations.proto      # Integration configurations
├── users.proto             # User management
├── triggers.proto          # Event trigger definitions
├── include/                # Shared proto definitions
└── private/                # Internal proto definitions
```

**Purpose**: Protocol Buffers define the API contracts. These are used to:
- Generate gRPC service code
- Generate REST API (via gRPC-Gateway)
- Generate client SDKs (TypeScript, Go)
- Generate OpenAPI specification

## Testing

### `test/` - Test Suite

```
test/
├── e2e/                    # End-to-end tests
│   ├── workflows/          # Workflow execution tests
│   ├── integrations/       # Integration tests
│   └── support/            # Test helpers and fixtures
├── fixtures/               # Test data and mocks
├── consumer/               # Consumer/producer tests
└── support/                # Test utilities
```

**Running tests**:
```bash
make test                   # Run all tests
make e2e                    # Run E2E tests
make test PKG_TEST_PACKAGES=./pkg/workers  # Run specific package tests
```

## Documentation

### `docs/` - Developer Documentation

```
docs/
├── GETTING_STARTED.md              # Development setup guide (start here!)
├── PROJECT_STRUCTURE.md            # This file
├── contributing/
│   ├── architecture.md             # System design and flow
│   ├── component-implementations.md # How to add new components
│   ├── component-design.md         # Component quality standards
│   ├── building-an-integration.md  # How to add new integrations
│   ├── e2e-tests.md                # Writing E2E tests
│   ├── pull-requests.md            # PR guidelines
│   ├── commit_sign-off.md          # DCO requirements
│   ├── quality.md                  # Code quality principles
│   ├── issue-tracking.md           # Issue management
│   └── ... (more guides)
├── design/                         # Design documentation
├── prd/                            # Product requirements
└── legal/                          # Legal documents
```

## Build Configuration

### Root Level Files

```
Makefile                   # All build commands (make dev.setup, etc.)
docker-compose.dev.yml     # Docker services for development
docker-compose.test.yml    # Docker services for testing
Dockerfile                 # Production image
docker-entrypoint.sh       # Container startup script
lint.toml                  # Linting configuration
go.mod                     # Go dependencies
package.json               # Node.js dependencies (in web_src/)
```

### Scripts

```
scripts/
├── vscode_run_tests.sh               # VSCode test runner
├── protoc.sh                         # Protobuf compilation
├── protoc_gateway.sh                 # gRPC-Gateway code generation
├── protoc_openapi_spec.sh            # OpenAPI spec generation
├── protoc_python.sh                  # Python protobuf generation
├── generate_components_docs.go       # Generate component documentation
├── check_go_coverage_budget.go       # Verify test coverage
├── verify_no_future_migrations.sh    # Migration validation
└── docker/                           # Docker-related scripts
```

## Release Management

### `release/` - Release Tools

```
release/
├── create_tag.sh                   # Create release tags
├── create-github-release.js        # Create GitHub releases
├── generate-sbom.sh                # Generate software bill of materials
├── superplane-image/               # Production container build
├── superplane-demo-image/          # Demo container build
├── superplane-helm-chart/          # Kubernetes Helm chart
└── superplane-single-host-tarball/  # Single-host deployment package
```

## Templates

### `templates/` - Workflow and Email Templates

```
templates/
├── canvases/               # Pre-built workflow templates
├── email/                  # Email templates
├── skills/                 # Reusable workflow skill templates
└── ...
```

## Architecture Layers

Understanding how SuperPlane layers work:

### Layer 1: API Interface (`pkg/grpc/`)

Handles incoming requests:
- gRPC services for internal communication
- REST endpoints for client access
- WebSocket for real-time updates

### Layer 2: Business Logic (`pkg/services/`)

Implements core functionality:
- User management
- Organization management
- Canvas/workflow management
- Component execution

### Layer 3: Data Access (`pkg/models/`)

Database interaction via GORM:
- All database models
- Query builders
- Relationships

### Layer 4: Integrations (`pkg/integrations/`)

External tool connections:
- API clients for third-party services
- Event adapters
- Credential management

### Layer 5: Background Workers (`pkg/workers/`)

Async processing:
- Event routing through workflows
- Component execution
- Webhook handling
- Scheduled tasks

## Data Flow Example: Workflow Execution

```
1. Webhook received (API endpoint)
   ↓
2. Event created in database (models.Event)
   ↓
3. Event queued for processing (workers queue)
   ↓
4. WorkflowEventRouter worker picks up event
   ↓
5. Router identifies next components to run (pkg/services)
   ↓
6. Component execution jobs queued
   ↓
7. ComponentExecutor workers run components
   ↓
8. Component results stored, new events created
   ↓
9. Repeat until workflow completes
```

## Key Files to Know

When starting development, these files are important:

- **[pkg/models/canvas.go](../pkg/models/canvas.go)** - Workflow definitions
- **[pkg/models/component.go](../pkg/models/component.go)** - Component setup
- **[pkg/workers/workflow_event_router.go](../pkg/workers/workflow_event_router.go)** - Event routing logic
- **[pkg/workers/component_executor.go](../pkg/workers/component_executor.go)** - Component execution
- **[cmd/server/main.go](../cmd/server/main.go)** - Server startup
- **[web_src/src/pages/workflowv2/](../web_src/src/pages/workflowv2/)** - Canvas UI
- **[AGENTS.md](../AGENTS.md)** - Coding standards and practices

## Related Documentation

- **[Architecture Overview](contributing/architecture.md)** - How systems interact
- **[Component Implementation](contributing/component-implementations.md)** - Building components
- **[Building Integrations](contributing/building-an-integration.md)** - Adding new integrations
- **[Getting Started](GETTING_STARTED.md)** - Development setup

---

**Need help?** Check out the full [Contributing Guide](../CONTRIBUTING.md) or join our [Discord](https://discord.gg/KC78eCNsnw)!
