import type { ComponentDefinition, ExtensionDefinition, IntegrationDefinition, ListResourcesInput, TriggerDefinition } from "../block-definitions.js";
import type { RuntimeValue } from "../runtime-context.js";
import { createRuntimeHarness, normalizeForJSON } from "../contexts/runtime-harness.js";
import type { DiscoveredOperation, OperationDescriptor, OperationHandler, RuntimeHarness } from "./types.js";

export function deriveOperations(definition: ExtensionDefinition): DiscoveredOperation[] {
  return collectOperations(definition).map(({ name, description }) => ({
    name,
    description,
  }));
}

export function collectOperations(definition: ExtensionDefinition): OperationDescriptor[] {
  const operations: OperationDescriptor[] = [];

  for (const integration of definition.integrations ?? []) {
    registerIntegrationOperations(operations, integration);
  }

  for (const component of definition.components ?? []) {
    registerComponentOperations(operations, component);
  }

  for (const trigger of definition.triggers ?? []) {
    registerTriggerOperations(operations, trigger);
  }

  return operations;
}

function registerIntegrationOperations(operations: OperationDescriptor[], integration: IntegrationDefinition): void {
  if (integration.sync) {
    operations.push(operation("integrations", integration.name, "sync", `${integration.label} sync`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await integration.sync?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (integration.cleanup) {
    operations.push(operation("integrations", integration.name, "cleanup", `${integration.label} cleanup`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await integration.cleanup?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (integration.handleAction) {
    operations.push(
      operation("integrations", integration.name, "handleAction", `${integration.label} handle action`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await integration.handleAction?.({
          name: invocation.actionName,
          configuration: invocation.configuration,
          parameters: invocation.parameters,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  if (integration.listResources) {
    operations.push(
      operation("integrations", integration.name, "listResources", `${integration.label} list resources`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await integration.listResources?.({
          configuration: invocation.configuration,
          input: invocation.input as unknown as ListResourcesInput,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result ?? []);
      }),
    );
  }

  if (integration.handleRequest) {
    operations.push(
      operation("integrations", integration.name, "handleRequest", `${integration.label} handle request`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await integration.handleRequest?.({
          configuration: invocation.configuration,
          body: invocation.body,
          headers: invocation.headers,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  const webhook = integration.webhook?.();
  if (!webhook) {
    return;
  }

  if (webhook.setup) {
    operations.push(
      operation("integrations", integration.name, "webhook.setup", `${integration.label} webhook setup`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await webhook.setup?.({
          runtime: harness.runtime,
          webhook: harness.webhook,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  if (webhook.cleanup) {
    operations.push(
      operation("integrations", integration.name, "webhook.cleanup", `${integration.label} webhook cleanup`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        await webhook.cleanup?.({
          runtime: harness.runtime,
          webhook: harness.webhook,
        });
        return finalizeInvocation(harness);
      }),
    );
  }

  if (webhook.compareConfig) {
    operations.push(
      operation(
        "integrations",
        integration.name,
        "webhook.compareConfig",
        `${integration.label} webhook compare config`,
        async (invocation) => {
          const harness = createRuntimeHarness(invocation);
          const result = await webhook.compareConfig?.({ current: invocation.current, requested: invocation.requested });
          return finalizeInvocation(harness, result ?? false);
        },
      ),
    );
  }

  if (webhook.merge) {
    operations.push(
      operation("integrations", integration.name, "webhook.merge", `${integration.label} webhook merge`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await webhook.merge?.({ current: invocation.current, requested: invocation.requested });
        return finalizeInvocation(harness, result);
      }),
    );
  }
}

function registerComponentOperations(operations: OperationDescriptor[], component: ComponentDefinition): void {
  if (component.setup) {
    operations.push(operation("components", component.name, "setup", `${component.label} setup`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await component.setup?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (component.processQueueItem) {
    operations.push(
      operation("components", component.name, "processQueueItem", `${component.label} process queue item`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await component.processQueueItem?.({
          configuration: invocation.configuration,
          input: invocation.input,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  operations.push(
    operation("components", component.name, "execute", `${component.label} execute`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await component.execute({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }),
  );

  if (component.handleAction) {
    operations.push(
      operation("components", component.name, "handleAction", `${component.label} handle action`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await component.handleAction?.({
          name: invocation.actionName,
          configuration: invocation.configuration,
          parameters: invocation.parameters,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  if (component.handleWebhook) {
    operations.push(
      operation("components", component.name, "handleWebhook", `${component.label} handle webhook`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await component.handleWebhook?.({
          configuration: invocation.configuration,
          body: invocation.body,
          headers: invocation.headers,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  if (component.cancel) {
    operations.push(operation("components", component.name, "cancel", `${component.label} cancel`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await component.cancel?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (component.cleanup) {
    operations.push(operation("components", component.name, "cleanup", `${component.label} cleanup`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await component.cleanup?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (component.onIntegrationMessage) {
    operations.push(
      operation(
        "components",
        component.name,
        "onIntegrationMessage",
        `${component.label} integration message`,
        async (invocation) => {
          const harness = createRuntimeHarness(invocation);
          const result = await component.onIntegrationMessage?.({
            configuration: invocation.configuration,
            message: invocation.message,
            runtime: harness.runtime,
          });
          return finalizeInvocation(harness, result);
        },
      ),
    );
  }
}

function registerTriggerOperations(operations: OperationDescriptor[], trigger: TriggerDefinition): void {
  if (trigger.setup) {
    operations.push(operation("triggers", trigger.name, "setup", `${trigger.label} setup`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await trigger.setup?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (trigger.handleAction) {
    operations.push(
      operation("triggers", trigger.name, "handleAction", `${trigger.label} handle action`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await trigger.handleAction?.({
          name: invocation.actionName,
          configuration: invocation.configuration,
          parameters: invocation.parameters,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  if (trigger.handleWebhook) {
    operations.push(
      operation("triggers", trigger.name, "handleWebhook", `${trigger.label} handle webhook`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await trigger.handleWebhook?.({
          configuration: invocation.configuration,
          body: invocation.body,
          headers: invocation.headers,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }

  if (trigger.cleanup) {
    operations.push(operation("triggers", trigger.name, "cleanup", `${trigger.label} cleanup`, async (invocation) => {
      const harness = createRuntimeHarness(invocation);
      await trigger.cleanup?.({
        configuration: invocation.configuration,
        input: invocation.input,
        runtime: harness.runtime,
      });
      return finalizeInvocation(harness);
    }));
  }

  if (trigger.onIntegrationMessage) {
    operations.push(
      operation("triggers", trigger.name, "onIntegrationMessage", `${trigger.label} integration message`, async (invocation) => {
        const harness = createRuntimeHarness(invocation);
        const result = await trigger.onIntegrationMessage?.({
          configuration: invocation.configuration,
          message: invocation.message,
          runtime: harness.runtime,
        });
        return finalizeInvocation(harness, result);
      }),
    );
  }
}

function finalizeInvocation(harness: RuntimeHarness, result?: unknown): RuntimeValue {
  return normalizeForJSON({
    result: normalizeForJSON(result),
    runtime: harness.snapshot(),
  });
}

function operation(
  blockType: "integrations" | "components" | "triggers",
  blockName: string,
  operationName: string,
  description: string,
  invoke: OperationHandler,
): OperationDescriptor {
  return {
    target: {
      blockType,
      blockName,
      operation: operationName,
    },
    name: `${blockType}.${blockName}.${operationName}`,
    description,
    invoke,
  };
}
