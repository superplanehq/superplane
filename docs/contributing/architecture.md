# Architecture Overview

This guide describes the main SuperPlane components and execution flow.

See the [architecture maps](../architecture/readme.md) for visual overviews.

SuperPlane uses a modular Go application. Startup flags select API and worker roles, so deployments can run these roles together or separately.
Fleet Manager and runners use separate executables.

## Key Concepts

### API

The public HTTP server exposes REST endpoints through gRPC-Gateway. Generated handlers call Go service implementations in the same process.
This path does not require a separate internal gRPC server. Protocol Buffers define the service contracts and generate the gateway and OpenAPI specification.
The server also handles authentication, webhooks, WebSocket connections, and file transfers through dedicated HTTP routes.

### Workers

Go workers handle workflow execution, scheduled hooks, integration setup, and maintenance. PostgreSQL stores their durable work records.
Core execution workers consume RabbitMQ notifications and periodically scan the database for unfinished work. Other workers, such as scheduled hook workers, poll PostgreSQL directly.

### Message Passing

RabbitMQ carries work notifications and application updates. Execution workers use these notifications to find durable records in PostgreSQL.
The event distributor also consumes notifications and sends updates to browser WebSocket connections. Runner gateways receive separate RabbitMQ notifications when runner state needs attention.

### Data Persistence

PostgreSQL stores application state, including organizations, canvases, integrations, work orders, events, queue items, and executions.
The application uses GORM for database access. The `migrate` CLI applies SQL files from `db/migrations/` and `db/data_migrations/`.

Blob storage holds files, artifacts, and archived runner logs. Supported providers include the filesystem, S3, GCS, and Azure.
Active runner logs use a separate filesystem store. A worker compresses completed logs and writes them to blob storage.
PostgreSQL stores the log lifecycle and storage references.

### Frontend

The frontend is a React application built with TypeScript and Vite. It communicates with the Public API via REST endpoints and receives real-time updates via WebSocket. The TypeScript API client is auto-generated from the OpenAPI specification.

### Runners

Runner components create tasks through the integrated backend, which stores encrypted task payloads in PostgreSQL.
Integrated runners register through HTTP, receive tasks through a WebSocket connection, and upload log chunks through HTTP.
Fleet Manager reads capacity through the installation admin HTTP API and provisions runners through AWS, Azure, GCP, or Docker.

## Core Event Processing System

The execution engine routes events through canvas graphs. PostgreSQL records each stage of this process.

**Event Lifecycle:**

1. **Event Creation**: Triggers and components create events in PostgreSQL with a `pending` state. A pending run can also start execution through the run initializer.

2. **Event Routing**: The event router locks a pending event and creates queue items for downstream nodes in a database transaction. After commit, it publishes RabbitMQ notifications.

3. **Queue Processing**: The node queue worker applies concurrency limits and processes queue items. Its default path creates an execution and marks the queue item as consumed.

4. **Component Execution**: The node executor calls the registered component. Components can complete immediately or use scheduled hooks, runner tasks, and webhooks to continue later.

5. **Event Propagation**: Component outputs create new events for downstream nodes. The run finalizer checks completed work and updates the run state.

Core execution workers also scan PostgreSQL to recover work when notifications are missed. Database transactions and locks coordinate concurrent workers.

## Authentication & Authorization

SuperPlane uses a multi-layered security model to authenticate users and enforce fine-grained permissions.

**Authentication:**

- **Browser Sessions**: Signed account cookies identify users of the web UI
- **Bearer Tokens**: API routes accept personal API tokens, organization API keys, and scoped tokens
- **Sign-in Providers**: Configured OAuth providers support account sign-in and account linking
- **Runner Credentials**: Runners exchange registration tokens for credentials that protect runner API routes

**Authorization:**

- **RBAC with Casbin**: Role-based access control is implemented using Casbin, which provides flexible policy management
- **Organization Permissions**: Resource permissions use the authenticated organization context
- **Permission Model**: Permissions use resource-action pairs, such as `canvases:create` and `integrations:read`
- **Groups and Roles**: Users can be assigned to groups with specific roles, enabling team-based access control

**Enforcement:**

- **API Layer**: HTTP middleware authenticates protected routes. Gateway authorization checks permissions before it calls service handlers
- **Context Propagation**: The server passes authenticated user and organization context to service handlers
- **Policies**: Casbin checks permissions against policies stored in PostgreSQL

Authentication depends on the route. Public pages, health checks, provider callbacks, installation administration, and runner connections have separate access rules.
Provider callbacks use their integration-specific validation rather than browser session authentication.

## Core Database Entities

The database model follows a hierarchical structure that enables multi-tenancy and resource organization:

**Account:**

- Represents a person's identity (email, name)
- Can have multiple OAuth providers (GitHub, Google, etc.) linked via `account_providers`
- One account can belong to multiple organizations through different users

**User:**

- Represents a person's membership in a specific organization
- Links an Account to an Organization
- Each user has organization-scoped permissions and roles
- A single account can have multiple users (one per organization they belong to)

**Organization:**

- Tenant boundary for application resources
- Canvases, integrations, secrets, and factory workspaces belong to an organization

**Factory:**

- Represents a workspace with work orders and automation lines
- Lines contain steps that reference apps, which are canvases
- A line dispatch creates canvas runs for its steps and records work order progress
- Uses the same execution engine as other canvases

**Canvas:**

- Versioned workflow graph, also called an app in factory views
- Belongs to an organization
- Can belong to a factory workspace
- Stores nodes, edges, node configurations, and metadata in canvas versions
- Versioning is always enabled: users edit the live canvas through a stage → commit loop. Canvas/console/files edits autosave into per-user `workflow_staged_files`; **Commit** creates a new version on main and promotes it to live
- Edit paths:
  - **Web UI and agent tools** stage canvas/console/files edits into `workflow_staged_files`, then require **Commit** with a message to go live. Agent reads serve the effective staged content and agent `patch_staging` writes go through the same staging layer
  - **CLI** stages via `PUT /staging` and goes live via `POST /staging/commit`
- Stale staging (when main moved while the user had staged edits) must be discarded before new edits can be committed.

**Integration:**

- Connects SuperPlane to external services (GitHub, Semaphore, etc.)
- Stores configuration and encrypted credentials
- Scoped to an organization
- Enables components to interact with third-party APIs

**Relationship Hierarchy:**

```
Account (1) ──→ (N) Users ──→ (1) Organization
Organization (1) ──→ (N) Canvases
Organization (1) ──→ (N) Factories ──→ (N) Work Orders
Factory (1) ──→ (N) Canvases
Organization (1) ──→ (N) Integrations
```

## Directory Structure

- **`cmd/`** - Application entry points

  - `cmd/server/` - Main server entry point
  - `cmd/cli/` - CLI tool entry point
  - `cmd/runner/` - Integrated runner entry point
  - `cmd/fleetmanager/` - Fleet Manager entry point

- **`pkg/`** - Core Go packages

  - `pkg/server/` - Application startup and role configuration
  - `pkg/grpc/` - Service implementations called by the HTTP gateway
  - `pkg/workers/` - Background worker processes
  - `pkg/models/` - Database models and ORM logic
  - `pkg/components/` - Workflow component implementations
  - `pkg/triggers/` - Event trigger implementations
  - `pkg/integrations/` - Third-party integration implementations (GitHub, Slack, Discord, etc.)
  - `pkg/public/` - Public HTTP server (serves UI + REST API)
  - `pkg/registry/` - Component and trigger registry
  - `pkg/authentication/` - User authentication logic
  - `pkg/authorization/` - RBAC and permission checks
  - `pkg/runners/` - Runner execution, protocol, and log storage
  - `pkg/fleets/` - Fleet Manager reconciliation and infrastructure providers
  - `pkg/blob/` - File and archived log storage providers

- **`web_src/`** - Frontend React application

  - `web_src/src/` - React pages, components, hooks, and shared libraries
  - `web_src/src/api-client/` - Generated TypeScript API client

- **`protos/`** - Protocol Buffer definitions for the API (workflows, components, integrations, users, etc.)

- **`db/`** - Database

  - `db/migrations/` - Database migration files
  - `db/structure.sql` - Current database schema

- **`test/`** - Test files

  - `test/e2e/` - End-to-end tests
  - `test/support/` - Test support utilities

- **`docs/`** - Documentation (concepts, contributing guides, examples)
