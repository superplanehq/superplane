# Architecture Overview

This document provides a high-level overview of SuperPlane's architecture to help contributors understand the core system components.

SuperPlane is built as a modular monolith, where each module (API, Workers, etc.) can be independently scaled based on workload requirements.

## Key Concepts

### API

The API handles all user requests and business logic. It consists of a Public API (serves the frontend and REST endpoints) and an Internal API (gRPC server with business logic). The API is built with Go, using gRPC for the Internal API and gRPC-Gateway to expose REST endpoints. API contracts are defined using Protocol Buffers.

### Workers

Workers are background processes that handle asynchronous tasks like workflow execution, webhook management, and maintenance. They communicate via RabbitMQ message queues. Workers are implemented in Go and consume messages from RabbitMQ queues for processing.

### Message Passing

Message passing enables asynchronous communication between the API and workers via RabbitMQ. It allows tasks to be queued and processed independently, enabling horizontal scaling and reliable message delivery. RabbitMQ provides the message queue infrastructure for this communication.

### Data Persistence

All data is persisted in PostgreSQL, including workflows, users, organizations, integrations, and components. The system uses GORM as the ORM for database access and migrations.

### Frontend

The frontend is a React application built with TypeScript and Vite. It communicates with the Public API via REST endpoints and receives real-time updates via WebSocket. The TypeScript API client is auto-generated from the OpenAPI specification.

## Core Event Processing System

The event processing system is the engine that drives workflow execution. It operates on an event-driven model where events flow through workflow graphs, triggering component execution.

**Event Lifecycle:**

1. **Event Creation**: Events are created when triggers fire (webhooks, schedules, manual starts) or when components complete execution. Events are stored in the database with a `pending` state.

2. **Event Routing**: The WorkflowEventRouter worker periodically scans for pending events and routes them through the workflow graph. It identifies downstream nodes connected via edges and creates queue items for nodes that should execute.

3. **Queue Processing**: Queue items are created for each node that should receive the event. These items are published to RabbitMQ for asynchronous processing.

4. **Component Execution**: Workers consume queue items and execute the corresponding components. Components process the event data, perform their actions, and emit new events upon completion.

5. **Event Propagation**: When a component finishes, it emits a new event that flows to its downstream nodes, continuing the workflow execution cycle.

This architecture enables parallel execution of independent workflow branches, reliable event delivery, and scalable processing through worker queues.

## Authentication & Authorization

SuperPlane uses a multi-layered security model to authenticate users and enforce fine-grained permissions.

**Authentication:**

- **JWT Tokens**: Users authenticate via JWT tokens stored in cookies (for web UI) or Bearer tokens (for API access)
- **User API Tokens**: Long-lived tokens for programmatic API access
- **OIDC Support**: Optional OIDC authentication for external identity providers

**Authorization:**

- **RBAC with Casbin**: Role-based access control is implemented using Casbin, which provides flexible policy management
- **Organization-Scoped Permissions**: All permissions are scoped to organizations, ensuring complete tenant isolation
- **Permission Model**: Permissions are defined as resource-action pairs (e.g., "workflows:create", "integrations:read")
- **Groups and Roles**: Users can be assigned to groups with specific roles, enabling team-based access control

**Enforcement:**

- **API Layer**: Authorization is enforced at the gRPC interceptor level, checking permissions before any business logic executes
- **Context Propagation**: User and organization context is extracted from request metadata and propagated through the call chain
- **Policy Loading**: Policies are loaded dynamically from the database, allowing real-time permission updates without service restarts

This architecture ensures that all API requests are authenticated and authorized before accessing any resources, with complete isolation between organizations.

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

- Top-level tenant boundary providing complete data isolation
- All resources (canvases, integrations, secrets) are scoped to an organization

**Canvas:**

- Workspace for building and managing workflows
- Belongs to an organization
- Contains multiple workflows with their nodes and edges
- Stores workflow graph structure, node configurations, and metadata

**Integration:**

- Connects SuperPlane to external services (GitHub, Semaphore, etc.)
- Stores encrypted credentials and configuration
- Scoped to an organization
- Enables components to interact with third-party APIs

**Relationship Hierarchy:**

```
Account (1) ──→ (N) Users ──→ (1) Organization
Organization (1) ──→ (N) Canvases
Organization (1) ──→ (N) Integrations
```

## Directory Structure

- **`cmd/`** - Application entry points

  - `cmd/server/` - Main server entry point
  - `cmd/cli/` - CLI tool entry point

- **`pkg/`** - Core Go packages

  - `pkg/grpc/` - gRPC service implementations
  - `pkg/workers/` - Background worker processes
  - `pkg/models/` - Database models and ORM logic
  - `pkg/components/` - Workflow component implementations
  - `pkg/triggers/` - Event trigger implementations
  - `pkg/applications/` - Third-party integration implementations (GitHub, Semaphore)
  - `pkg/public/` - Public HTTP server (serves UI + REST API)
  - `pkg/registry/` - Component and trigger registry
  - `pkg/authentication/` - User authentication logic
  - `pkg/authorization/` - RBAC and permission checks

- **`web_src/`** - Frontend React application

  - `web_src/src/` - React source code (pages, components, hooks, utils)
  - `web_src/src/api/` - Auto-generated TypeScript API client

- **`protos/`** - Protocol Buffer definitions for the API (workflows, components, integrations, users, etc.)

- **`db/`** - Database

  - `db/migrations/` - Database migration files
  - `db/structure.sql` - Current database schema

- **`test/`** - Test files

  - `test/e2e/` - End-to-end tests
  - `test/support/` - Test support utilities

- **`docs/`** - Documentation (concepts, contributing guides, examples)
