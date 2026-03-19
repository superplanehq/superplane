import type { RuntimeValue } from "../context/runtime-value.js";

export interface ExecutionStateEmission {
  channel: string;
  payloadType: string;
  payloads: RuntimeValue[];
}

export interface ExecutionStateEffects {
  finished: boolean;
  passed: boolean;
  failed: { reason: string; message: string } | null;
  kv: Record<string, string>;
  emissions: ExecutionStateEmission[];
}
