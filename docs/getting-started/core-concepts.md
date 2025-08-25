# Core Concepts

## Organizations

Organizations provide the top-level boundary for all Superplane resources. Each organization operates as an isolated tenant with complete separation of data and users.

**Key characteristics:**
- **Data isolation**: Complete separation between organizations - no shared resources or data access
- **User management**: Organization admins control member access and permissions
- **Resource container**: All Canvases, integrations, and workflows exist within an organization

**Practical usage:**
Organizations typically represent companies, business units, or major project divisions. Choose your organization scope based on data access requirements and administrative boundaries.

## Canvases

Canvas is your main workspace for building and managing DevOps workflows. Each Canvas operates as a self-contained project environment where you connect tools, build automation, and monitor execution.

**Primary functions:**
- **Tool integration**: Connect external services (source control, CI/CD platforms, deployment tools) through integrations
- **Workflow construction**: Build multiple workflows on a single Canvas using visual component connections
- **Real-time monitoring**: Observe current state of all workflows, component status, and active executions
- **Component inspection**: Click any component to view its execution history, event logs, and configuration details in the Canvas sidebar

**Canvas workspace includes:**
- **Integration management**: Configure and maintain connections to external DevOps tools
- **Workflow builder**: Visual interface for connecting components into automated processes  
- **Execution dashboard**: Real-time view of workflow status, component states, and running processes
- **Component sidebar**: Detailed view of individual component data including event history, run history, and configuration
- **Secrets management**: Encrypted storage for API keys, tokens, and sensitive configuration data
- **User permissions**: Canvas-level access control for team collaboration

**Technical architecture:**
Canvas provides execution isolation - workflows, data, and state are contained within each Canvas boundary. This allows teams to separate concerns by application, environment, or process domain while maintaining unified tool integrations.

### Integrations

Integrations connect external DevOps tools to your Canvas, enabling components to interact with their APIs and receive webhook events.

**Technical function:**
- **API connectivity**: Establish authenticated connections to external tool APIs
- **Webhook registration**: Configure external tools to send event notifications to Superplane
- **Credential management**: Store and manage authentication tokens, API keys, and OAuth credentials
- **Component enablement**: Unlock integration-specific components that can perform tool operations

**Integration lifecycle:**
1. **Authentication setup**: Configure OAuth flows or API token authentication with the target tool
2. **Permission verification**: Confirm Superplane has necessary API permissions for intended operations
3. **Webhook configuration**: Set up external tool to send relevant events to Canvas webhook endpoints
4. **Component availability**: Integration-specific components become available in the Canvas component library

**Current integrations:**
- **Source control platforms**: Repository monitoring, branch operations, pull request management
- **CI/CD systems**: Pipeline triggering, build status monitoring, deployment coordination

**Technical requirements:**
- Valid API credentials with appropriate permissions for the target tool
- Network connectivity between Superplane and external tool APIs
- Webhook endpoint accessibility for receiving external tool events

### Secrets

Secrets provide encrypted storage for sensitive data that components need during workflow execution. All secrets are scoped to the Canvas level with encryption at rest and in transit.

**Technical implementation:**
- **Encrypted storage**: AES-256 encryption for all secret values with secure key management
- **Canvas scoping**: Secrets are isolated per Canvas - no cross-Canvas access possible
- **Runtime injection**: Secret values are securely injected into component execution environments
- **Access logging**: All secret usage is logged for audit and compliance tracking

**Common secret types:**
- **API tokens**: Authentication tokens for external service APIs
- **OAuth credentials**: Client IDs, client secrets, and refresh tokens for OAuth integrations
- **Database credentials**: Connection strings, usernames, and passwords for database access
- **Infrastructure keys**: SSH keys, certificate files, and cloud provider credentials
- **Custom configuration**: Environment-specific settings, endpoints, and feature flags

**Usage in components:**
Components reference secrets by name during configuration. At runtime, Superplane resolves secret names to actual values and injects them into the execution context. Secret values are never logged or exposed in component outputs.

**Security characteristics:**
- **Zero-knowledge access**: Secret values are encrypted and only decrypted during component execution
- **Role-based access**: Canvas permissions control who can create, modify, or delete secrets
- **Audit trail**: Complete logging of secret creation, modification, and usage for compliance
- **Automatic cleanup**: Unused secrets can be identified through usage tracking

## Components

### Overview

[Content placeholder]

### Events

[Content placeholder]

### Executors

[Content placeholder]

### Inputs and Outputs

[Content placeholder]

### Runs

[Content placeholder]

### Stages

[Content placeholder]

## Workflows

[Content placeholder]