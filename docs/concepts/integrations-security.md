# Integrations and Security

## Table of Contents

- [Overview](#overview)
- [Integrations](#integrations)
  - [What Integrations Provide](#what-integrations-provide)
  - [Creating an Integration](#creating-an-integration)
  - [Using Integrations in Stages](#using-integrations-in-stages)
  - [Creating Event Sources Through Integrations](#creating-event-sources-through-integrations)
  - [Supported Integration Types](#supported-integration-types)
- [Secrets](#secrets)
  - [Creating Secrets](#creating-secrets)
  - [Using Secrets in Components](#using-secrets-in-components)
  - [Security Characteristics](#security-characteristics)

---

## Overview

Integrations connect external DevOps tools to your Canvas, enabling components to interact with their APIs and receive webhook events. Secrets provide encrypted storage for sensitive data that components need during workflow execution.

Together, integrations and secrets enable secure, authenticated connections to external systems while keeping credentials protected and workflows automated.

---

## Integrations

Integrations are connections between SuperPlane and external services that enable two-way communication: SuperPlane can make API calls to external tools, and external tools can send events to SuperPlane workflows.

### What Integrations Provide

**API connectivity:** Establish authenticated connections to external tool APIs for triggering operations, reading data, and managing resources.

**Webhook registration:** Configure external tools to send event notifications to SuperPlane when important events occur (builds complete, deployments finish, alerts fire).

**Credential management:** Store and manage authentication tokens, API keys, and OAuth credentials securely at the integration level.

**Component enablement:** Unlock integration-specific components and executors that can perform operations on the connected tool.

### Creating an Integration

Integrations are defined in YAML and specify how to authenticate with external services:

```yaml
apiVersion: v1
kind: Integration
metadata:
  name: github-integration
  canvasId: canvas-123
spec:
  # Type determines what SuperPlane does when using this integration
  type: github
  
  # Base URL for the service
  url: https://github.com/myorg
  
  # Authentication configuration
  auth:
    use: AUTH_TYPE_TOKEN
    token:
      valueFrom:
        secret:
          name: github-credentials
          key: token
```

**Integration lifecycle:**
1. **Authentication setup** - Configure API token authentication with the target tool
2. **Permission verification** - Confirm SuperPlane has necessary API permissions
3. **Webhook configuration** - External tool sends relevant events to Canvas webhook endpoints
4. **Component availability** - Integration-specific components become available

### Using Integrations in Stages

Reference integrations in stage executors to perform operations on external systems:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: trigger-build
spec:
  executor:
    type: semaphore
    integration:
      name: semaphore-integration
    resource:
      type: project
      name: my-project
    spec:
      branch: main
      pipelineFile: .semaphore/build.yml
      parameters:
        VERSION: ${{ inputs.VERSION }}
```

### Creating Event Sources Through Integrations

Integrations can automatically provision event sources on external systems:

```yaml
apiVersion: v1
kind: EventSource
metadata:
  name: repo-events
  canvasId: canvas-123
spec:
  integration:
    name: github-integration
  resource:
    type: repository
    name: my-repository
```

When you create this event source, SuperPlane automatically:
- Creates webhooks on the GitHub repository
- Configures the webhook to send events to SuperPlane
- Handles authentication and webhook verification

### Supported Integration Types

#### GitHub Integration

Connect to GitHub repositories to trigger GitHub Actions workflows and receive repository events.

**Required permissions:** GitHub fine-grained personal access token (PAT) with:
- Actions - Read and Write
- Webhooks - Read and Write

```yaml
apiVersion: v1
kind: Integration
metadata:
  name: github-integration
  canvasId: canvas-123
spec:
  type: github
  url: https://github.com/myorg
  auth:
    use: AUTH_TYPE_TOKEN
    token:
      valueFrom:
        secret:
          name: github-token
          key: token
```

**Capabilities:**
- Trigger GitHub Actions workflows via workflow dispatch
- Receive webhook events for pushes, pull requests, releases
- Access repository metadata and commit information

#### Semaphore Integration

Connect to Semaphore CI/CD to trigger pipelines and receive build notifications.

**Required authentication:** Personal API token or service account token with appropriate project permissions.

```yaml
apiVersion: v1
kind: Integration
metadata:
  name: semaphore-integration
  canvasId: canvas-123
spec:
  type: semaphore
  url: https://myorg.semaphoreci.com
  auth:
    use: AUTH_TYPE_TOKEN
    token:
      valueFrom:
        secret:
          name: semaphore-credentials
          key: token
```

**Capabilities:**
- Trigger Semaphore workflows and tasks
- Receive pipeline status notifications
- Access build logs and artifacts
- Monitor project and workflow status

---

## Secrets

Secrets provide encrypted storage for sensitive data like API tokens, database credentials, and configuration values. All secrets are scoped to the Canvas level with encryption at rest and in transit.

### Creating Secrets

Create secrets that will be managed by SuperPlane:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-credentials
  canvasId: canvas-123
spec:
  provider: local
  local:
    data:
      api-key: "sk_live_abc123def456"
      database-url: "postgresql://user:pass@host:5432/db"
      webhook-secret: "whsec_xyz789"
```

**Secret structure:**
- `provider: local` - Currently the only supported provider
- `local.data` - Key-value pairs of secret data
- Keys and values are both encrypted when stored

### Using Secrets in Components

Reference secrets in component configurations to provide secure access to credentials:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: deploy-app
spec:
  # Define secrets available to this component
  secrets:
    - name: DEPLOY_TOKEN
      valueFrom:
        secret:
          name: api-credentials
          key: api-key
    - name: DB_URL
      valueFrom:
        secret:
          name: api-credentials
          key: database-url

  executor:
    type: http
    spec:
      url: https://api.example.com/deploy
      headers:
        Authorization: "Bearer ${{ secrets.DEPLOY_TOKEN }}"
      payload:
        database_url: ${{ secrets.DB_URL }}
        version: ${{ inputs.VERSION }}
```

**Usage patterns:**
- **Integration authentication** - Store API tokens for external service access
- **Database credentials** - Connection strings and authentication details
- **Webhook signatures** - Secret keys for validating incoming webhooks
- **Custom configuration** - Environment-specific settings and feature flags

### Security Characteristics

**Encryption:** AES-256 encryption for all secret values with secure key management. Keys are encrypted at rest and only decrypted during component execution.

**Canvas scoping:** Secrets are isolated per Canvas with no cross-Canvas access possible. Each Canvas maintains its own encrypted secret store.

**Runtime injection:** Secret values are securely injected into component execution environments without being logged or exposed in component outputs.

**Zero-knowledge access:** Secret values are encrypted and only decrypted during component execution. SuperPlane administrators cannot view secret contents.

**Role-based access:** Canvas permissions control who can create, modify, or delete secrets. Only authorized users can manage secret configurations.