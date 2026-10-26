import type {
  FactoriesFactoryCustomVerification,
  FactoriesFactoryCustomVerificationSettings,
  FactoriesFactoryCustomVerificationStep,
  FactoryCustomVerificationStartMode,
} from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";

export const CUSTOM_VERIFICATION_TEMPLATE_ID = "custom-verification";

export type CustomVerificationMode = "manual" | "automatic";
export type ManualCustomVerification = {
  id: string;
  name: string;
  icon?: string;
};

export const DEFAULT_CUSTOM_VERIFICATION_ICON = "sparkles";

export function customVerificationMode(value: FactoryCustomVerificationStartMode | undefined): CustomVerificationMode {
  return value === "START_MODE_AUTOMATIC" ? "automatic" : "manual";
}

export function customVerificationModeCopy(mode: CustomVerificationMode): {
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

export function customVerificationDraft(verification?: FactoriesFactoryCustomVerification): PlanningReviewDraft {
  const settings = verification?.settings;
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
    title: verification?.automation?.name?.trim() || "Custom automation",
    components: [
      {
        id: "custom-verification-agent",
        title: "Automation agent",
        description: "Run the configured automation steps.",
        expanded: true,
        component: agent?.component,
        configuration: {
          ...customVerificationAgentConfiguration(agent),
          steps,
        },
        concurrency: { max: "5", key: "" },
      },
    ],
  };
}

function customVerificationAgentConfiguration(
  agent: FactoriesFactoryCustomVerificationSettings["agent"],
): Record<string, string> {
  return {
    model: agent?.model ?? "",
    credentialSource: agent?.credentialSource ?? "",
    credentialIntegrationName: agent?.credentialIntegrationName ?? "",
  };
}

export function customVerificationSettings(
  mode: CustomVerificationMode,
  draft: PlanningReviewDraft,
  includeAgent: boolean,
  icon = DEFAULT_CUSTOM_VERIFICATION_ICON,
): FactoriesFactoryCustomVerificationSettings {
  const component = draft.components[0];
  const configuration = component?.configuration ?? {};
  const steps = Array.isArray(configuration.steps) ? (configuration.steps as PlanningReviewStep[]) : [];
  return {
    startMode: mode === "automatic" ? "START_MODE_AUTOMATIC" : "START_MODE_MANUAL",
    icon,
    agent:
      includeAgent && component?.component
        ? {
            component: component.component,
            model: typeof configuration.model === "string" ? configuration.model : "",
            credentialSource: typeof configuration.credentialSource === "string" ? configuration.credentialSource : "",
            credentialIntegrationName:
              typeof configuration.credentialIntegrationName === "string"
                ? configuration.credentialIntegrationName
                : "",
          }
        : undefined,
    steps: steps.map(stepToApi),
  };
}

function stepFromApi(step: FactoriesFactoryCustomVerificationStep): PlanningReviewStep {
  return {
    name: step.name ?? "",
    type: step.type === "TYPE_BASH" ? "bash" : "prompt",
    prompt: step.prompt,
    command: step.command,
    workingDirectory: step.workingDirectory,
    secrets: step.secrets,
  };
}

function stepToApi(step: PlanningReviewStep): FactoriesFactoryCustomVerificationStep {
  return {
    name: step.name,
    type: step.type === "bash" ? "TYPE_BASH" : "TYPE_PROMPT",
    prompt: step.type === "prompt" ? (step.prompt ?? "") : undefined,
    command: step.type === "bash" ? (step.command ?? "") : undefined,
    workingDirectory: step.workingDirectory,
    secrets: step.type === "bash" ? step.secrets : undefined,
  };
}
