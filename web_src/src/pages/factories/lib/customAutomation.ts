import type {
  FactoriesFactoryCustomAutomation,
  FactoriesFactoryCustomAutomationSettings,
  FactoriesFactoryCustomAutomationStep,
  FactoryCustomAutomationStartMode,
} from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";

export const CUSTOM_AUTOMATION_TEMPLATE_ID = "custom-verification";

export type CustomAutomationMode = "manual" | "automatic";
export type ManualCustomAutomation = {
  id: string;
  name: string;
  icon?: string;
};

export const DEFAULT_CUSTOM_AUTOMATION_ICON = "sparkles";
export const DEFAULT_CUSTOM_AUTOMATION_TIMEOUT_SECONDS = 3600;

export function customAutomationMode(value: FactoryCustomAutomationStartMode | undefined): CustomAutomationMode {
  return value === "START_MODE_AUTOMATIC" ? "automatic" : "manual";
}

export function customAutomationModeCopy(mode: CustomAutomationMode): {
  trigger: string;
  action: string;
} {
  if (mode === "automatic") {
    return {
      trigger: "On Verify and new commits",
      action: "Run automation automatically",
    };
  }
  return {
    trigger: "Manual start on a Verify task",
    action: "Run automation on demand",
  };
}

export function workspaceAgentFromOnboarding(onboarding?: {
  agentHarness?: string;
}): FactoriesFactoryCustomAutomationSettings["agent"] | undefined {
  switch (onboarding?.agentHarness) {
    case "AGENT_HARNESS_SUPERPLANE":
      return { component: "runnerSuperPlane" };
    case "AGENT_HARNESS_CODEX":
      return { component: "runnerCodex" };
    case "AGENT_HARNESS_CLAUDE_CODE":
      return { component: "runnerClaudeCode" };
    default:
      return undefined;
  }
}

export function customAutomationDraft(automation?: FactoriesFactoryCustomAutomation): PlanningReviewDraft {
  const settings = automation?.settings;
  const agent = settings?.agent;
  const steps = settings
    ? (settings.steps ?? []).map(stepFromApi)
    : [
        {
          name: "Hello world",
          type: "bash" as const,
          command: 'echo "hello world"',
          workingDirectory: "repo",
        },
      ];
  return {
    title: automation?.automation?.name?.trim() || "Custom automation",
    components: [
      {
        id: "custom-automation-agent",
        title: "Automation agent",
        description: "Run the configured automation steps.",
        expanded: true,
        component: agent?.component,
        configuration: {
          ...customAutomationAgentConfiguration(agent),
          steps,
          executionTimeoutSeconds: customAutomationTimeout(settings?.executionTimeoutSeconds),
        },
        concurrency: { max: "5", key: "" },
      },
    ],
  };
}

function customAutomationAgentConfiguration(
  agent: FactoriesFactoryCustomAutomationSettings["agent"],
): Record<string, string> {
  return {
    model: agent?.model ?? "",
    credentialSource: agent?.credentialSource ?? "",
    credentialIntegrationName: agent?.credentialIntegrationName ?? "",
  };
}

export function customAutomationSettings(
  mode: CustomAutomationMode,
  draft: PlanningReviewDraft,
  icon = DEFAULT_CUSTOM_AUTOMATION_ICON,
): FactoriesFactoryCustomAutomationSettings {
  const component = draft.components[0];
  const configuration = component?.configuration ?? {};
  const steps = Array.isArray(configuration.steps) ? (configuration.steps as PlanningReviewStep[]) : [];
  const credentialSource = typeof configuration.credentialSource === "string" ? configuration.credentialSource : "";
  const credentialIntegrationName =
    typeof configuration.credentialIntegrationName === "string" ? configuration.credentialIntegrationName : "";
  const superPlaneAgent = component?.component === "runnerSuperPlane";
  return {
    startMode: mode === "automatic" ? "START_MODE_AUTOMATIC" : "START_MODE_MANUAL",
    icon,
    executionTimeoutSeconds: customAutomationTimeout(configuration.executionTimeoutSeconds),
    agent: component?.component
      ? {
          component: component.component,
          model: typeof configuration.model === "string" ? configuration.model : "",
          ...(!superPlaneAgent && credentialSource
            ? { credentialSource, credentialIntegrationName: credentialIntegrationName || undefined }
            : {}),
        }
      : undefined,
    steps: steps.map(stepToApi),
  };
}

function customAutomationTimeout(value: unknown): number {
  return typeof value === "number" && value > 0 ? value : DEFAULT_CUSTOM_AUTOMATION_TIMEOUT_SECONDS;
}

function stepFromApi(step: FactoriesFactoryCustomAutomationStep): PlanningReviewStep {
  return {
    name: step.name ?? "",
    type: step.type === "TYPE_BASH" ? "bash" : "prompt",
    prompt: step.prompt,
    command: step.command,
    workingDirectory: step.workingDirectory,
    secrets: step.secrets,
  };
}

function stepToApi(step: PlanningReviewStep): FactoriesFactoryCustomAutomationStep {
  return {
    name: step.name,
    type: step.type === "bash" ? "TYPE_BASH" : "TYPE_PROMPT",
    prompt: step.type === "prompt" ? (step.prompt ?? "") : undefined,
    command: step.type === "bash" ? (step.command ?? "") : undefined,
    workingDirectory: step.workingDirectory,
    secrets: step.type === "bash" ? step.secrets : undefined,
  };
}
