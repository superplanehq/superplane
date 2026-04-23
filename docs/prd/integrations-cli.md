## Setting up an integration, interactively

You can set up an integration through the CLI, interactively, with `integrations setup init --interactive`:

```
superplane integrations setup init --interactive \
  --name rtx \
  --integration semaphore 
```

This one will create the integration, and start the setup flow, showing instructions and inputs appropriately. It will be done to completion. Good for use for humans.

## Setting up an integration, non-interactively

It is also possible to set up an integration, non-interactively. This is particular useful when using the SuperPlane with an LLM agent. To start things up, you use `setup init` without the `--interactive` flag:

```
superplane integrations setup init \
  --name rtx \
  --integration semaphore
```

This creates the integration, and sets its initial state, showing instructions and inputs in the output.

To submit the next step, you use `setup submit`:

```
superplane integrations setup submit \
  --name rtx \
  --integration semaphore \
  --step-name <step-name> \
  --step-inputs '...'
```

If for some reason, you started and came back to the setup flow later, you can find the setup status with `setup status`:

```
superplane integrations setup status --name rtx --integration semaphore
```

This will give you the current status of the setup flow, including the instructions and inputs for the next step.