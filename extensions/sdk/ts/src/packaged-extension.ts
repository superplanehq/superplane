import {
  type InvokeExtensionJob,
  type InvocationOutput,
  type InvocationPayload,
} from "@superplanehq/runtime";
import type { ExtensionDefinition } from "./block-definitions.js";
import {
  deriveManifest,
  validateExtensionDefinition,
} from "./runtime/manifest.js";
import {
  collectOperations,
  deriveOperations,
  runInvokeExtensionJob,
} from "./runtime/operations.js";
import type { ExtensionDiscovery } from "./runtime/types.js";

export interface ExtensionRuntimeModule {
  manifest: ExtensionDiscovery["manifest"];
  operations: ExtensionDiscovery["operations"];
  run(job: InvokeExtensionJob): Promise<InvocationOutput>;
}

export { deriveManifest, deriveOperations, validateExtensionDefinition };

export function discoverExtension(
  definition: ExtensionDefinition,
): ExtensionDiscovery {
  return {
    manifest: deriveManifest(definition),
    operations: deriveOperations(definition),
  };
}

export function createPackagedExtensionRuntime(
  definition: ExtensionDefinition,
): ExtensionRuntimeModule {
  return createRuntimeModule(definition);
}

export function createRuntimeModule(
  definition: ExtensionDefinition,
): ExtensionRuntimeModule {
  const discovery = discoverExtension(definition);
  const operationIndex = new Map(
    collectOperations(definition).map((operation) => [
      operation.name,
      operation,
    ]),
  );

  return {
    manifest: discovery.manifest,
    operations: discovery.operations,
    run(job: InvokeExtensionJob): Promise<InvocationOutput> {
      return runInvokeExtensionJob(operationIndex, job);
    },
  };
}

export async function invokePackagedExtension(
  definition: ExtensionDefinition,
  payload: InvocationPayload,
): Promise<InvocationOutput> {
  return createRuntimeModule(definition).run({
    type: "invoke-extension",
    payload,
  });
}
