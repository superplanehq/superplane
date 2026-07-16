import type { ConfigurationField, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

export type LiveNodePreRunStatusPurpose = "setup" | "runtime";

export type LiveNodePreRunStatus = {
  title: string;
  description?: string;
  purpose: LiveNodePreRunStatusPurpose;
};

type NodeActivityData = {
  executions: unknown[];
  events: unknown[];
};

const ACTIVE_LIVE_RUNTIME_EXECUTION_STATES = new Set(["STATE_STARTED", "STATE_PENDING"]);
const LIVE_INTERACTIVE_SIDEBAR_COMPONENTS = new Set(["approval"]);

function hasActiveLiveRuntimeExecution(executions: unknown[]): boolean {
  return executions.some((execution) => {
    const state = (execution as { state?: string }).state;
    return Boolean(state && ACTIVE_LIVE_RUNTIME_EXECUTION_STATES.has(state));
  });
}

type ResolveLiveNodePreRunStatusOptions = {
  configurationFields?: ConfigurationField[];
};

export function resolveLiveNodePreRunStatus(
  workflowNode: ComponentsNode,
  nodeData: NodeActivityData,
  options: ResolveLiveNodePreRunStatusOptions = {},
): LiveNodePreRunStatus {
  const { configurationFields } = options;
  const isPlaceholder = !workflowNode.component && workflowNode.name === "New Component";
  if (isPlaceholder) {
    return { title: "Continue editing to choose a component", purpose: "setup" };
  }

  if (workflowNode.errorMessage) {
    return {
      title: "Finish configuring this component",
      description: formatLiveNodeConfigurationIssue(workflowNode.errorMessage, configurationFields),
      purpose: "setup",
    };
  }

  const nodeType = workflowNode.type || "TYPE_ACTION";
  if (nodeType === "TYPE_TRIGGER") {
    if (nodeData.events.length === 0) {
      return { title: "Waiting for the first event", purpose: "runtime" };
    }

    return { title: "Inspect activity in Runs", purpose: "runtime" };
  }

  if (nodeData.executions.length === 0) {
    return { title: "Waiting for the first run...", purpose: "runtime" };
  }

  if (
    workflowNode.component &&
    LIVE_INTERACTIVE_SIDEBAR_COMPONENTS.has(workflowNode.component) &&
    hasActiveLiveRuntimeExecution(nodeData.executions)
  ) {
    return {
      title: "Action required",
      description: "Use the controls below to continue.",
      purpose: "runtime",
    };
  }

  return { title: "Inspect activity in Runs", purpose: "runtime" };
}

export function formatLiveNodeConfigurationIssue(
  errorMessage: string,
  configurationFields?: ConfigurationField[],
): string {
  const trimmed = errorMessage.trim();
  if (!trimmed) {
    return trimmed;
  }

  const requiredMatch = trimmed.match(/^field '([^']+)' is required$/i);
  if (requiredMatch) {
    const fieldName = requiredMatch[1];
    return `${getConfigurationFieldLabel(fieldName, configurationFields)} is required`;
  }

  const fieldErrorMatch = trimmed.match(/^field '([^']+)': (.+)$/i);
  if (fieldErrorMatch) {
    const [, fieldName, issue] = fieldErrorMatch;
    return `${getConfigurationFieldLabel(fieldName, configurationFields)}: ${issue}`;
  }

  const integrationRequiredMatch = trimmed.match(/^integration is required for (.+)$/i);
  if (integrationRequiredMatch) {
    return "Connect an integration instance to continue";
  }

  return trimmed;
}

function getConfigurationFieldLabel(fieldName: string, configurationFields?: ConfigurationField[]): string {
  const field = configurationFields?.find((candidate) => candidate.name === fieldName);
  if (field?.label?.trim()) {
    return field.label.trim();
  }

  return humanizeFieldName(fieldName);
}

function humanizeFieldName(fieldName: string): string {
  const withSpaces = fieldName
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .trim();

  if (!withSpaces) {
    return fieldName;
  }

  return withSpaces.charAt(0).toUpperCase() + withSpaces.slice(1);
}
