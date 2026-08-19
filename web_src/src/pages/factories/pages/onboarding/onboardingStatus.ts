import type { FactoriesFactory, FactoriesUpdateFactoryOnboardingBody } from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import { WIZARD_STEPS, type IssuesChoiceId, type WizardStepId } from "./onboardingFixtures";

export function initialOnboardingSelections(onboarding: FactoriesFactory["onboarding"]): IntegrationSelections {
  const selections: IntegrationSelections = {};
  if (onboarding?.vcsIntegrationId) {
    selections.github = {
      id: onboarding.vcsIntegrationId,
      name: onboarding.vcsIntegrationId,
      ready: false,
    };
  }
  if (onboarding?.agentIntegrationId) {
    selections.claude = {
      id: onboarding.agentIntegrationId,
      name: onboarding.agentIntegrationId,
      ready: false,
    };
  }
  return selections;
}

export function localIssuesSource(source?: string): IssuesChoiceId | null {
  const sources: Record<string, IssuesChoiceId> = {
    ISSUES_SOURCE_VCS: "vcs",
    ISSUES_SOURCE_LINEAR: "linear",
    ISSUES_SOURCE_JIRA: "jira",
    ISSUES_SOURCE_SKIP: "skip",
  };
  return source ? (sources[source] ?? null) : null;
}

export function apiIssuesSource(source: IssuesChoiceId | null): FactoriesUpdateFactoryOnboardingBody["issuesSource"] {
  const sources: Partial<Record<IssuesChoiceId, FactoriesUpdateFactoryOnboardingBody["issuesSource"]>> = {
    vcs: "ISSUES_SOURCE_VCS",
    linear: "ISSUES_SOURCE_LINEAR",
    jira: "ISSUES_SOURCE_JIRA",
    skip: "ISSUES_SOURCE_SKIP",
  };
  return source ? sources[source] : "ISSUES_SOURCE_UNSPECIFIED";
}

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
