export type RuntimeOperation = "component.execute" | "component.setup";

export type ComponentExecutionRequest = {
  operation: RuntimeOperation;
  component: string;
  context: {
    executionId: string;
    workflowId: string;
    organizationId: string;
    nodeId: string;
    sourceNodeId: string;
    configuration: unknown;
    integrationConfiguration?: Record<string, unknown>;
    data: unknown;
    metadata?: Record<string, unknown>;
    nodeMetadata?: Record<string, unknown>;
  };
};

export type ComponentExecutionOutcome = "pass" | "fail" | "noop";

export type ComponentOutput = {
  channel: string;
  payloadType: string;
  payload: unknown;
};

export type ComponentExecutionKV = {
  key: string;
  value: string;
};

export type ComponentExecutionResponse = {
  outcome: ComponentExecutionOutcome;
  errorReason?: string;
  error?: string;
  outputs?: ComponentOutput[];
  metadata?: Record<string, unknown>;
  nodeMetadata?: Record<string, unknown>;
  kvs?: ComponentExecutionKV[];
};

export type SetupContext = {
  configuration: unknown;
  integrationConfiguration?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  nodeMetadata?: Record<string, unknown>;
};

export type ExecuteContext = ComponentExecutionRequest["context"] & {
  logger: RuntimeLogger;
};

export type RuntimeLogger = {
  debug(message: string, fields?: Record<string, unknown>): void;
  info(message: string, fields?: Record<string, unknown>): void;
  warn(message: string, fields?: Record<string, unknown>): void;
  error(message: string, fields?: Record<string, unknown>): void;
};

export type ActionImplementation = {
  setup?(ctx: SetupContext): Promise<void> | void;
  execute(ctx: ExecuteContext): Promise<ComponentExecutionResponse> | ComponentExecutionResponse;
};
