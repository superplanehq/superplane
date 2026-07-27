import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasNodeQueueItem,
  ConfigurationField,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";

export type RunNodeDetailTabKey = "details" | "payload" | "configuration";

export type RunNodeDetailTabAvailability = {
  hasDetailsSection: boolean;
  hasPayload: boolean;
  hasConfig: boolean;
};

export type RunNodeDetailTabData = {
  details?: Record<string, unknown>;
  payload?: unknown;
  configuration?: unknown;
};

export type RunInspectorOutputSection = {
  channel: string;
  value: unknown;
  sizeKb: string;
};

export type RunInspectorCurrentUser = {
  id: string;
  email: string;
  roles?: string[];
  groups?: string[];
};

export type RunInspectorApprovalRecord = {
  index: number;
  state?: string;
  type?: string;
  user?: {
    id?: string;
    email?: string;
    name?: string;
  };
  roleRef?: {
    name?: string;
    displayName?: string;
  };
  groupRef?: {
    name?: string;
    displayName?: string;
  };
};

export type RunInspectorNodeActions = {
  canStop: boolean;
  canPushThrough: boolean;
  approvalRecords: RunInspectorApprovalRecord[];
};

export type RunInspectorUpstreamSection = {
  nodeId: string;
  nodeName: string;
  workflowNode?: ComponentsNode;
  badge: { badgeColor: string; label: string } | null;
  output: unknown;
};

export type RunInspectorNodeSection = {
  sectionValue: string;
  nodeId: string;
  nodeName: string;
  workflowNode?: ComponentsNode;
  execution?: CanvasesCanvasNodeExecution;
  executionRef?: CanvasesCanvasNodeExecutionRef;
  queueItem?: CanvasesCanvasNodeQueueItem;
  isTrigger: boolean;
  isQueued: boolean;
  createdAt?: string;
  durationMs?: number;
  badge: { badgeColor: string; label: string } | null;
  tabData: RunNodeDetailTabData | null;
  upstreamSections: RunInspectorUpstreamSection[];
  primaryInputNodeId?: string;
  outputSections: RunInspectorOutputSection[];
  errorMessage?: string;
  actions: RunInspectorNodeActions;
  configurationFields: ConfigurationField[];
};

export type RunInspectorErrorSummary = {
  nodeId: string;
  nodeName: string;
  message: string;
};
