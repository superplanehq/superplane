- [Management and usage](#management-and-usage)
  - [Creating an integration](#creating-an-integration)
  - [Using an integration in a stage](#using-an-integration-in-a-stage)
  - [Creating an event source through an integration](#creating-an-event-source-through-an-integration)
- [Supported Integration Types](#supported-integration-types)
  - [Semaphore](#semaphore)
  - [GitHub](#github)

---

Integrations are a way to connect Superplane to external services.

An integration:
- Configures how Superplane should authenticate and make requests to an external service, so it can read and create resources in it.
- Configures how Superplane should receive requests from the external service.
- Once created, it is used to create stage executions.
- It can be used to create event sources for specific resources in the external service.

## Management and usage

### Creating an integration

This is what an integration looks like in YAML:

```yaml
kind: Integration
metadata:
  name: semaphore-integration
  canvasId: canvas-123
spec:

  #
  # Type of the integration.
  # This field would determine what SuperPlane
  # actually does when this integration is used on a stage or event source.
  #
  # For example, when an event source for a Semaphore integration is created, 
  # SuperPlane needs to create a Semaphore notification to monitor the result of executions.
  # For a GitHub integration, SuperPlane would create the webhook, ...
  #
  type: semaphore

  #
  # URL where the integration lives.
  # For some cases, just the type is enough,
  # But for some, like Semaphore, where the organization is part of the URL,
  # we need to specify the URL too.
  #
  url: https://semaphore.semaphoreci.com

  #
  # Specifies the authentication used for making requests to the integration.
  #
  auth:

    #
    # Any type of bearer token (personal API token, service account token, ...).
    #
    use: AUTH_TYPE_TOKEN
    token:
      valueFrom:
        secret:
          name: semaphore
          key: token

    #
    # The integration accepts OIDC Connect ID tokens issued by SuperPlane.
    # For example, we can use this for AWS.
    #
    # use: oidc
```

### Using an integration in a stage

When creating stages, you must specify the stage executor, and in the executor, you should reference an integration. For example, this is how you run a Semaphore workflow through the `semaphore-integration` integration:

```yaml
executor:
  type: semaphore
  integration:
    name: semaphore-integration
  resource:
    type: project
    name: semaphore-demo-go
  spec:
    branch: main
    pipelineFile: .semaphore/semaphore.yml
    parameters: {}
```

### Creating an event source through an integration

You can create an event source through the integration, and SuperPlane will take care of provisioning everything on the integration side. For example, here's how you can create an event source for a Semaphore project:

```yaml
apiVersion: v1
kind: EventSource
metadata:
  name: my-project
  canvasId: a1787a2e-dba7-42d0-8431-31dbf0252b92
spec:
  integration:
    name: semaphore-integration
  resource:
    type: project
    name: semaphore-demo-go
```

## Supported Integration Types

### Semaphore

The Semaphore integration allows you to connect to Semaphore CI/CD and trigger workflows or tasks in specific Semaphore projects, and receive notifications about pipeline status updates. For authentication, you must use a personal API token or service account token.

Here's an example of a Semaphore integration:

```yaml
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
          name: semaphore
          key: token
```

### GitHub

The GitHub integration allows you to connect to GitHub repositories and trigger GitHub Actions workflows.

For authentication, you must use a GitHub fine-grained personal access token (PAT) with appropriate permissions:
- Actions - Read and Write
- Webhooks - Read and Write

Here's an example of a GitHub integration:

```yaml
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
