export type StepStatus = "passed" | "created" | "success" | "running" | "failed" | "pending";
export type ComponentProvider = "github" | "slack" | "superplane";

export type StepDetail = {
  startedAt?: string;
  finishedAt?: string;
  duration?: string;
  agent?: string;
  attempt?: string;
  output?: string;
  logs?: string[];
  inputs?: string[];
};

export type NodePorts = {
  target?: boolean;
  out?: boolean;
  side?: boolean;
};

export type StepNodeData = {
  /** Human step label (sidebar / instance name). */
  title: string;
  /** Component type shown in the node header with the provider icon. */
  componentName: string;
  provider: ComponentProvider;
  status: StepStatus;
  kind: "step" | "action";
  detail?: string;
  meta?: string[];
  showLogs?: boolean;
  details: StepDetail;
  /** Which connection handles are in use (derived from edges). */
  ports?: NodePorts;
};
