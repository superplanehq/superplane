import type { FactoriesFactory } from "@/api-client";

import { WIZARD_STEPS, type WizardStepId } from "./onboardingFixtures";

export function isFactoryOnboardingComplete(factory: FactoriesFactory | null | undefined): boolean {
  return Boolean(factory?.onboarding?.completedAt);
}

export function isWizardStepId(value: string | null | undefined): value is WizardStepId {
  return WIZARD_STEPS.some((step) => step.id === value);
}

// The API sends every onboarding field, so a question the user did not answer
// yet arrives as the zero value of its enum, not as an absent field.
function answered(value: string | undefined): boolean {
  return Boolean(value) && !value?.endsWith("_UNSPECIFIED");
}

// Setup resumes at the first wizard step that still needs an answer.
// Name comes after the agent step; a new workspace still has a placeholder
// name until the user confirms it there.
export function initialWizardStep(onboarding: FactoriesFactory["onboarding"]): WizardStepId {
  if (answered(onboarding?.agentHarness)) return "name";
  if (answered(onboarding?.issuesSource)) return "agent";
  if (onboarding?.appRepository) return "issues";
  if (onboarding?.vcsIntegrationId) return "repo";
  return "vcs";
}
