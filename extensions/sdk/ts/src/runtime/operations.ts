import type {
  ComponentDefinition,
  ExtensionDefinition,
} from "../block-definitions.js";
import {
  createRuntimeHarness,
  normalizeForJSON,
} from "../contexts/runtime-harness.js";
import type {
  ComponentCancelInvocationEnvelope,
  ComponentCancelInvocationOutput,
  ComponentExecuteInvocationEnvelope,
  ComponentExecuteInvocationOutput,
  ComponentHandleActionInvocationEnvelope,
  ComponentHandleActionInvocationOutput,
  ComponentHandleWebhookInvocationEnvelope,
  ComponentHandleWebhookInvocationOutput,
  ComponentOnIntegrationMessageInvocationEnvelope,
  ComponentOnIntegrationMessageInvocationOutput,
  ComponentSetupInvocationEnvelope,
  ComponentSetupInvocationOutput,
  ComponentInvocationTarget,
  DiscoveredOperation,
  InvocationEnvelope,
  InvocationOutput,
  OperationDescriptor,
  OperationHandler,
  RuntimeHarness,
} from "./types.js";

export function deriveOperations(
  definition: ExtensionDefinition,
): DiscoveredOperation[] {
  return collectOperations(definition).map(({ name, description }) => ({
    name,
    description,
  }));
}

export function collectOperations(
  definition: ExtensionDefinition,
): OperationDescriptor[] {
  const operations: OperationDescriptor[] = [];

  for (const component of definition.components ?? []) {
    registerComponentOperations(operations, component);
  }

  return operations;
}

function registerComponentOperations(
  operations: OperationDescriptor[],
  component: ComponentDefinition,
): void {
  if (component.setup) {
    operations.push(
      operation(
        "components",
        component.name,
        "setup",
        `${component.label} setup`,
        async (invocation) => {
          const setupInvocation =
            invocation as ComponentSetupInvocationEnvelope;
          const harness = createRuntimeHarness(invocation);
          await component.setup?.({
            configuration: setupInvocation.context.configuration,
            context: harness.context,
          });
          return finalizeInvocation(setupInvocation, harness);
        },
      ),
    );
  }

  operations.push(
    operation(
      "components",
      component.name,
      "execute",
      `${component.label} execute`,
      async (invocation) => {
        const executionInvocation =
          invocation as ComponentExecuteInvocationEnvelope;
        const harness = createRuntimeHarness(invocation);
        await component.execute({
          configuration: executionInvocation.context.configuration,
          data: executionInvocation.invocation.data,
          context: harness.context,
        });
        return finalizeInvocation(executionInvocation, harness);
      },
    ),
  );

  if (component.handleAction) {
    operations.push(
      operation(
        "components",
        component.name,
        "handleAction",
        `${component.label} handle action`,
        async (invocation) => {
          const actionInvocation =
            invocation as ComponentHandleActionInvocationEnvelope;
          const harness = createRuntimeHarness(invocation);
          const result = await component.handleAction?.({
            name: actionInvocation.invocation.name,
            configuration: actionInvocation.context.configuration,
            parameters: actionInvocation.invocation.parameters,
            context: harness.context,
          });
          return finalizeInvocation(actionInvocation, harness, result);
        },
      ),
    );
  }

  if (component.handleWebhook) {
    operations.push(
      operation(
        "components",
        component.name,
        "handleWebhook",
        `${component.label} handle webhook`,
        async (invocation) => {
          const webhookInvocation =
            invocation as ComponentHandleWebhookInvocationEnvelope;
          const harness = createRuntimeHarness(invocation);
          const result = await component.handleWebhook?.({
            configuration: webhookInvocation.context.configuration,
            body: webhookInvocation.invocation.body,
            headers: webhookInvocation.invocation.headers,
            context: harness.context,
          });
          return finalizeInvocation(webhookInvocation, harness, result);
        },
      ),
    );
  }

  if (component.cancel) {
    operations.push(
      operation(
        "components",
        component.name,
        "cancel",
        `${component.label} cancel`,
        async (invocation) => {
          const cancelInvocation =
            invocation as ComponentCancelInvocationEnvelope;
          const harness = createRuntimeHarness(invocation);
          await component.cancel?.({
            configuration: cancelInvocation.context.configuration,
            data: cancelInvocation.invocation.data,
            context: harness.context,
          });
          return finalizeInvocation(cancelInvocation, harness);
        },
      ),
    );
  }

  if (component.onIntegrationMessage) {
    operations.push(
      operation(
        "components",
        component.name,
        "onIntegrationMessage",
        `${component.label} integration message`,
        async (invocation) => {
          const messageInvocation =
            invocation as ComponentOnIntegrationMessageInvocationEnvelope;
          const harness = createRuntimeHarness(invocation);
          const result = await component.onIntegrationMessage?.({
            configuration: messageInvocation.context.configuration,
            message: messageInvocation.invocation.message,
            context: harness.context,
          });
          return finalizeInvocation(messageInvocation, harness, result);
        },
      ),
    );
  }
}

function finalizeInvocation(
  invocation: InvocationEnvelope,
  harness: RuntimeHarness,
  result?: unknown,
): InvocationOutput {
  const effects = harness.snapshot();

  switch (invocation.target.operation) {
    case "setup":
      return {
        target: invocation.target,
        effects,
      } satisfies ComponentSetupInvocationOutput;
    case "execute":
      return {
        target: invocation.target,
        effects,
      } satisfies ComponentExecuteInvocationOutput;
    case "handleAction":
      return {
        target: invocation.target,
        effects,
      } satisfies ComponentHandleActionInvocationOutput;
    case "handleWebhook":
      return {
        target: invocation.target,
        effects,
        response: normalizeForJSON(result),
      } satisfies ComponentHandleWebhookInvocationOutput;
    case "cancel":
      return {
        target: invocation.target,
        effects,
      } satisfies ComponentCancelInvocationOutput;
    case "onIntegrationMessage":
      return {
        target: invocation.target,
        effects,
      } satisfies ComponentOnIntegrationMessageInvocationOutput;
  }
}

function operation(
  blockType: ComponentInvocationTarget["blockType"],
  blockName: string,
  operationName: ComponentInvocationTarget["operation"],
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
