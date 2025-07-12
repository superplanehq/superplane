## Integrations

Integrations are a way to connect Superplane to external services.

An integration:
- Configures how Superplane should authenticate and make requests to the external service, so it can read/create resources in it
- Configures how Superplane should receive requests from the external service, for example, execution outputs.
- Once configured, can be used as an event source directly, or be used as the source for more specialized event sources.

## Management and usage

### Creating an integration

This is what an integration looks like in YAML:

```yaml
kind: Integration
metadata:
  name: semaphore-integration

  #
  # If canvasId is specified, the integration will be scoped to that canvas.
  #
  # canvasId: canvas-123
spec:

  #
  # Type of the integration.
  # This field would determine what SuperPlane
  # actually does when an integration of this type is created.
  #
  # For example, when a Semaphore integration is created,
  # SuperPlane needs to create a Semaphore notification to monitor the result of executions.
  # For a GitHub integration, SuperPlane would create the webhook, ...
  #
  type: TYPE_SEMAPHORE

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

  #
  # Specifies whether or not the integration supports OIDC ID tokens,
  # and can use them to authenticate requests sent to SuperPlane.
  #
  # If it does, the integration can also specify which claims it accepts.
  #
  oidc:
    enabled: true
    claims: {}
```

### Using an integration in a stage executor

```yaml
executor:
  type: TYPE_SEMAPHORE
  integration:
    domain: ORGANIZATION
    name: semaphore-integration
  semaphore:
    projectId: f5808f38-bc99-4b11-9edd-6969d4664802
    branch: main
    pipelineFile: .semaphore/semaphore.yml
    parameters: {}
```

### Using an integration as an event source

You can create a more specialized event source from it, filtering only the events you want to receive:

```yaml
apiVersion: v1
kind: EventSource
metadata:
  name: my-specific-project
  canvasId: a1787a2e-dba7-42d0-8431-31dbf0252b92
spec:
  integration:
    domain: ORGANIZATION
    name: semaphore-integration
  semaphore:
    project: semaphore-demo-go
  # github:
  #   repository: semaphore-demo-go
```

## GitHub

callback URL: ???
 - not using this one for now, because we are using the OAuth already
 - we probably should migrate to use just a GitHub app, instead of a GitHub app and a OAuth app

setup URL:
  http://localhost:8000/integrations/github/app_installation?state=<org-id>

The setup URL receives a `installation_id` parameter, but since that can be spoofed, we shouldn't use that.

Permissions
  - Actions (read/write) -> needed to run GitHub Actions workflows
  - Webhooks (read/write) -> needed to receive updates about specific repositories
Events
  - We only need events about app installations
