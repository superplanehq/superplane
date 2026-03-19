import type {
  InvocationEnvelope,
  InvocationOutput,
  InvocationTarget,
} from "@superplanehq/runtime";
import type { ManifestJSONValue, ManifestV1 } from "../manifest-schema.js";

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

export type OperationHandler = (
  envelope: InvocationEnvelope,
) => Promise<InvocationOutput>;

export interface OperationDescriptor {
  target: InvocationTarget;
  name: string;
  description: string;
  invoke: OperationHandler;
}
