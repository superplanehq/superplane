## Factory

A factory is responsible for completing work orders, by dispatching them through lines.
A factory can define multiple lines.

```yaml
kind: Factory
metadata:
  name: SuperPlane Factory
spec:
  lines:
    - name: bug
      steps:
        - type: runApp
          app:
            name: Factory
            entrypoint: start-implementation
        - type: runApp
          app:
            name: Factory
            entrypoint: run-verifications
    - name: poc
      steps:
        - type: runApp
          app:
            name: POC builder
            entrypoint: create-storybook
        - type: runApp
          app:
            name: POC builder
            entrypoint: deploy-storybook
```

Users can dispatch work orders to any line with:

```bash
superplane factory dispatch --order order-123 --line bugs
```

### Factory owned apps

Apps can be either owned by a factory, or not. If an app is owned by a factory, it gains new components to use in its canvas, to manage work orders.

## Work orders

A work order is a single unit of work that can be dispatched to a factory line.

Work orders can by sourced from external systems, through an app, or manually created.
