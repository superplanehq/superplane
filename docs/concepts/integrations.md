## Integrations

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

When creating stages, you must specify the stage executor, and in the executor, you should reference an integration. For example, this is how you run a Semaphore workflow during the stage execution, through the `semaphore-integration` integration:

```yaml
executor:
  type: TYPE_SEMAPHORE
  integration:
    name: semaphore-integration
  resource:
    type: project
    name: semaphore-demo-go
  semaphore:
    branch: main
    pipelineFile: .semaphore/semaphore.yml
    parameters: {}
```

NOTE: not all executor types require an integration, but most of them do. For example, the HTTP executor does not require an integration to be used.

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
