export interface FieldOption {
  label: string;
  value: string;
}

export interface FieldDefinition {
  name: string;
  label: string;
  type: "string" | "text" | "number" | "bool" | "select" | "object";
  description?: string;
  required?: boolean;
  default?: unknown;
  options?: FieldOption[];
}

export interface ActionDefinition<TParams = Record<string, unknown>> {
  label: string;
  description?: string;
  fields: Record<string, Omit<FieldDefinition, "name">>;
  execute: (
    params: TParams,
    ctx: ExecutionContext,
  ) => Promise<Record<string, unknown>>;
}

export interface ExecutionContext {
  input: unknown;
}

export interface ActionManifest {
  name: string;
  label: string;
  description: string;
  fields: Array<{
    name: string;
    label: string;
    type: string;
    description: string;
    required: boolean;
    default?: unknown;
    options?: FieldOption[];
  }>;
}

export interface Manifest {
  name: string;
  label: string;
  icon: string;
  description: string;
  actions: ActionManifest[];
}

export interface PluginOptions {
  name: string;
  label?: string;
  icon?: string;
  description?: string;
}

export interface ExecuteRequest {
  parameters: Record<string, unknown>;
  input?: unknown;
}

export interface ExecuteResponse {
  success: boolean;
  data?: Record<string, unknown>;
  error?: string;
}
