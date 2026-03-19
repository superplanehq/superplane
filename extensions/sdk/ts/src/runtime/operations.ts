import type {
  ComponentDefinition,
  ExtensionDefinition,
} from "../block-definitions.js";
import {
  createInvokeExtensionRuntime,
  normalizeForJSON,
  normalizeInvocationEnvelope,
  type ComponentCancelInvocationEnvelope,
  type ComponentCancelInvocationOutput,
  type ComponentExecuteInvocationEnvelope,
  type ComponentExecuteInvocationOutput,
  type ComponentHandleActionInvocationEnvelope,
  type ComponentHandleActionInvocationOutput,
  type ComponentHandleWebhookInvocationEnvelope,
  type ComponentHandleWebhookInvocationOutput,
  type ComponentInvocationTarget,
  type ComponentOnIntegrationMessageInvocationEnvelope,
  type ComponentOnIntegrationMessageInvocationOutput,
  type ComponentSetupInvocationEnvelope,
  type ComponentSetupInvocationOutput,
  type InvokeExtensionJob,
  type InvokeExtensionRuntime,
  type InvocationEnvelope,
  type InvocationOutput,
} from "@superplanehq/runtime";
import type {
  DiscoveredOperation,
  OperationDescriptor,
  OperationHandler,
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
          const runtime = createInvokeExtensionRuntime(invocation);
          await component.setup?.({
            configuration: setupInvocation.context.configuration,
            context: runtime.context,
          });
          return finalizeInvocation(setupInvocation, runtime);
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
        const runtime = createInvokeExtensionRuntime(invocation);
        await component.execute({
          configuration: executionInvocation.context.configuration,
          data: executionInvocation.invocation.data,
          context: runtime.context,
        });
        return finalizeInvocation(executionInvocation, runtime);
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
          const runtime = createInvokeExtensionRuntime(invocation);
          const result = await component.handleAction?.({
            name: actionInvocation.invocation.name,
            configuration: actionInvocation.context.configuration,
            parameters: actionInvocation.invocation.parameters,
            context: runtime.context,
          });
          return finalizeInvocation(actionInvocation, runtime, result);
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
          const runtime = createInvokeExtensionRuntime(invocation);
          const result = await component.handleWebhook?.({
            configuration: webhookInvocation.context.configuration,
            body: webhookInvocation.invocation.body,
            headers: webhookInvocation.invocation.headers,
            context: runtime.context,
          });
          return finalizeInvocation(webhookInvocation, runtime, result);
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
          const runtime = createInvokeExtensionRuntime(invocation);
          await component.cancel?.({
            configuration: cancelInvocation.context.configuration,
            data: cancelInvocation.invocation.data,
            context: runtime.context,
          });
          return finalizeInvocation(cancelInvocation, runtime);
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
          const runtime = createInvokeExtensionRuntime(invocation);
          const result = await component.onIntegrationMessage?.({
            configuration: messageInvocation.context.configuration,
            message: messageInvocation.invocation.message,
            context: runtime.context,
          });
          return finalizeInvocation(messageInvocation, runtime, result);
        },
      ),
    );
  }
}

function finalizeInvocation(
  invocation: InvocationEnvelope,
  runtime: InvokeExtensionRuntime,
  result?: unknown,
): InvocationOutput {
  const effects = runtime.snapshot();

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

export async function runInvokeExtensionJob(
  operationIndex: Map<string, OperationDescriptor>,
  job: InvokeExtensionJob,
): Promise<InvocationOutput> {
  if (job.type !== "invoke-extension") {
    throw new Error(`Unsupported job type ${job.type}`);
  }

  return invokeOperation(operationIndex, job.payload);
}

async function invokeOperation(
  operationIndex: Map<string, OperationDescriptor>,
  payload: InvokeExtensionJob["payload"],
): Promise<InvocationOutput> {
  const operationName = formatOperationName(payload.target);
  const operation = operationIndex.get(operationName);
  if (!operation) {
    throw new Error(`Operation ${operationName} is not registered`);
  }

  return operation.invoke(normalizeInvocationEnvelope(payload));
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

function formatOperationName(
  target: InvokeExtensionJob["payload"]["target"],
): string {
  return `${target.blockType}.${target.blockName}.${target.operation}`;
}
