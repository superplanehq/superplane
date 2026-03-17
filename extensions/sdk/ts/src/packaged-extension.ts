import { normalizeInvocationEnvelope } from "./contexts/runtime-harness.js";
import type { ExtensionDefinition } from "./block-definitions.js";
import type { RuntimeValue } from "./runtime-context.js";
import { deriveManifest, validateExtensionDefinition } from "./runtime/manifest.js";
import { collectOperations, deriveOperations } from "./runtime/operations.js";
import type { ExtensionDiscovery, InvocationPayload, OperationDescriptor } from "./runtime/types.js";

export interface PackagedExtensionRuntime {
  manifest: ExtensionDiscovery["manifest"];
  operations: ExtensionDiscovery["operations"];
  invoke(payload: InvocationPayload): Promise<RuntimeValue>;
}

export { deriveManifest, deriveOperations, validateExtensionDefinition };

export function discoverExtension(definition: ExtensionDefinition): ExtensionDiscovery {
  return {
    manifest: deriveManifest(definition),
    operations: deriveOperations(definition),
  };
}

export function createPackagedExtensionRuntime(definition: ExtensionDefinition): PackagedExtensionRuntime {
  const discovery = discoverExtension(definition);
  const operationIndex = new Map(collectOperations(definition).map((operation) => [operation.name, operation]));

  return {
    manifest: discovery.manifest,
    operations: discovery.operations,
    invoke(payload: InvocationPayload): Promise<RuntimeValue> {
      return invokeOperation(operationIndex, payload);
    },
  };
}

export async function invokePackagedExtension(definition: ExtensionDefinition, payload: InvocationPayload): Promise<RuntimeValue> {
  return createPackagedExtensionRuntime(definition).invoke(payload);
}

async function invokeOperation(operationIndex: Map<string, OperationDescriptor>, payload: InvocationPayload): Promise<RuntimeValue> {
  const operationName = formatOperationName(payload.target);
  const operation = operationIndex.get(operationName);
  if (!operation) {
    throw new Error(`Operation ${operationName} is not registered`);
  }

  return operation.invoke(normalizeInvocationEnvelope(payload));
}

function formatOperationName(target: InvocationPayload["target"]): string {
  return `${target.blockType}.${target.blockName}.${target.operation}`;
}
