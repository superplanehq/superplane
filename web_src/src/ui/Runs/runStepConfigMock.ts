import type { ConfigurationField } from "@/api-client";

/**
 * Prototype-only mock configuration schemas + values for the run panel's
 * "Step configuration" mode. The run panel does not receive the real component
 * catalog, and the storybook fixtures use invented component names, so we supply
 * representative schemas here purely to demonstrate the read-only configuration
 * view. Production would resolve `configurationFields` from the real catalog
 * (`allComponentsByName` / `allTriggersByName`).
 */

export interface MockStepConfig {
  fields: ConfigurationField[];
  values: Record<string, unknown>;
}

const select = (options: string[]): ConfigurationField["typeOptions"] => ({
  select: { options: options.map((value) => ({ value, label: value })) },
});

const MOCK_STEP_CONFIG: Record<string, MockStepConfig> = {
  github: {
    fields: [
      { name: "repository", label: "Repository", type: "string" },
      { name: "event", label: "Event", type: "select", typeOptions: select(["push", "pull_request", "release"]) },
      { name: "branch", label: "Branch", type: "string" },
    ],
    values: { repository: "acme/superplane", event: "push", branch: "refs/heads/main" },
  },
  deploy: {
    fields: [
      {
        name: "environment",
        label: "Environment",
        type: "select",
        typeOptions: select(["staging", "production"]),
      },
      {
        name: "strategy",
        label: "Strategy",
        type: "select",
        typeOptions: select(["rolling", "recreate", "canary"]),
      },
      { name: "replicas", label: "Replicas", type: "number" },
      { name: "healthCheckPath", label: "Health check path", type: "string" },
      { name: "autoRollback", label: "Auto rollback", type: "boolean" },
    ],
    values: {
      environment: "production",
      strategy: "rolling",
      replicas: 3,
      healthCheckPath: "/healthz",
      autoRollback: true,
    },
  },
  notify: {
    fields: [
      { name: "channel", label: "Channel", type: "select", typeOptions: select(["slack", "email", "pagerduty"]) },
      { name: "target", label: "Target", type: "string" },
      { name: "message", label: "Message", type: "text" },
      { name: "notifyOnFailureOnly", label: "Notify on failure only", type: "boolean" },
    ],
    values: {
      channel: "slack",
      target: "#deploys",
      message: "Deployment {{ run.name }} finished with status {{ run.result }}.",
      notifyOnFailureOnly: false,
    },
  },
  test: {
    fields: [
      { name: "suite", label: "Suite", type: "select", typeOptions: select(["unit", "integration", "e2e"]) },
      { name: "parallelism", label: "Parallelism", type: "number" },
      { name: "retries", label: "Retries", type: "number" },
      { name: "failFast", label: "Fail fast", type: "boolean" },
    ],
    values: { suite: "integration", parallelism: 4, retries: 2, failFast: true },
  },
  build: {
    fields: [
      { name: "dockerfile", label: "Dockerfile", type: "string" },
      { name: "context", label: "Build context", type: "string" },
      { name: "target", label: "Target stage", type: "string" },
      { name: "pushImage", label: "Push image", type: "boolean" },
    ],
    values: { dockerfile: "Dockerfile", context: ".", target: "production", pushImage: true },
  },
  approval: {
    fields: [
      { name: "approvers", label: "Approvers", type: "string" },
      { name: "minApprovals", label: "Minimum approvals", type: "number" },
      { name: "timeoutMinutes", label: "Timeout (minutes)", type: "number" },
      { name: "instructions", label: "Instructions", type: "text" },
    ],
    values: {
      approvers: "platform-team",
      minApprovals: 1,
      timeoutMinutes: 60,
      instructions: "Confirm the staging smoke tests passed before approving.",
    },
  },
};

const DEFAULT_CONFIG: MockStepConfig = {
  fields: [
    { name: "action", label: "Action", type: "string" },
    { name: "timeoutSeconds", label: "Timeout (seconds)", type: "number" },
    { name: "continueOnError", label: "Continue on error", type: "boolean" },
  ],
  values: { timeoutSeconds: 300, continueOnError: false },
};

/** Returns representative read-only configuration (schema + values) for a component. */
export function getMockStepConfig(component?: string): MockStepConfig {
  const preset = component ? MOCK_STEP_CONFIG[component] : undefined;
  if (preset) {
    return preset;
  }
  return {
    fields: DEFAULT_CONFIG.fields,
    values: { ...DEFAULT_CONFIG.values, action: component ?? "" },
  };
}
