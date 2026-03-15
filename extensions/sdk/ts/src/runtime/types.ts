import type { ManifestJSONValue, ManifestV1 } from "../manifest-schema.js";
import type { RuntimeContext, RuntimeValue, WebhookContext } from "../runtime-context.js";

export interface InvocationTarget {
  blockType: "integrations" | "components" | "triggers";
  blockName: string;
  operation: string;
}

export interface InvocationPayload {
  target: InvocationTarget;
  configuration?: RuntimeValue;
  input?: RuntimeValue;
  current?: RuntimeValue;
  requested?: RuntimeValue;
  parameters?: Record<string, RuntimeValue>;
  actionName?: string;
  headers?: Record<string, string[]>;
  body?: RuntimeValue;
  message?: RuntimeValue;
  integration?: {
    id?: string;
    configuration?: Record<string, RuntimeValue>;
    metadata?: RuntimeValue;
  };
  webhook?: {
    id?: string;
    url?: string;
    secret?: RuntimeValue;
    metadata?: RuntimeValue;
    configuration?: RuntimeValue;
  };
  metadata?: RuntimeValue;
}

export interface DiscoveredOperation {
  name: string;
  description?: string;
  inputSchema?: ManifestJSONValue;
  outputSchema?: ManifestJSONValue;
}

export interface ExtensionDiscovery {
  manifest: ManifestV1;
  operations: DiscoveredOperation[];
}

export type OperationHandler = (envelope: InvocationEnvelope) => Promise<RuntimeValue>;

export interface OperationDescriptor {
  target: InvocationTarget;
  name: string;
  description: string;
  invoke: OperationHandler;
}

export interface InvocationEnvelope {
  target: InvocationTarget;
  configuration: RuntimeValue;
  input: RuntimeValue;
  current: RuntimeValue;
  requested: RuntimeValue;
  parameters: Record<string, RuntimeValue>;
  actionName: string;
  headers: Record<string, string[]>;
  body: Uint8Array;
  message: RuntimeValue;
  integration: {
    id: string;
    configuration: Record<string, RuntimeValue>;
    metadata: RuntimeValue;
  };
  webhook: {
    id: string;
    url: string;
    secret: Uint8Array;
    metadata: RuntimeValue;
    configuration: RuntimeValue;
  };
  metadata: RuntimeValue;
}

export interface RuntimeHarness {
  runtime: RuntimeContext;
  webhook: WebhookContext;
  snapshot(): RuntimeValue;
}
