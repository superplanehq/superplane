import type { FactoriesFactory, FactoriesUpdateFactoryOnboardingBody, OrganizationsIntegration } from "@/api-client";
import { selectionFromInstance, type IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import { AGENT_PROVIDER_IDS, type AgentProviderId } from "./onboardingAgentReadiness";
import { WIZARD_STEPS, type IssuesChoiceId, type VcsHostId, type WizardStepId } from "./onboardingFixtures";

// The connection list restores a saved installation. GitHub is included so a
// reload can mark that installation ready. It is also manual: the wizard must
// not adopt a different GitHub connection that already exists in the organization.
export const ONBOARDING_CONNECTION_NAMES = ["github", "bitbucket", "jira", "linear", ...AGENT_PROVIDER_IDS] as const;

// Agent keys stay unselected so a new workspace can use the SuperPlane template
// when hosted credit is available.
export const ONBOARDING_MANUAL_CONNECTION_NAMES = ["github", ...AGENT_PROVIDER_IDS] as const;

/** The saved Git host of the workspace. An empty provider means GitHub. */
export function onboardingVcsHost(onboarding: FactoriesFactory["onboarding"]): VcsHostId | null {
  if (!onboarding?.vcsIntegrationId) return null;
  return onboarding.vcsProvider === "bitbucket" ? "bitbucket" : "github";
}

export function initialOnboardingSelections(onboarding: FactoriesFactory["onboarding"]): IntegrationSelections {
  const selections: IntegrationSelections = {};
  const vcsHost = onboardingVcsHost(onboarding);
  if (vcsHost && onboarding?.vcsIntegrationId) {
    selections[vcsHost] = {
      id: onboarding.vcsIntegrationId,
      name: onboarding.vcsIntegrationId,
      ready: false,
    };
  }
  const agentProvider = savedAgentProvider(onboarding);
  const agentIntegrationId = onboarding?.agentIntegrationId?.trim() ?? "";
  if (agentProvider && agentIntegrationId) {
    selections[agentProvider] = {
      id: agentIntegrationId,
      name: agentIntegrationId,
      ready: false,
    };
  }
  return selections;
}

function savedAgentProvider(onboarding: FactoriesFactory["onboarding"]): AgentProviderId | null {
  if (!onboarding?.agentIntegrationId?.trim()) return null;
  switch (onboarding.agentHarness) {
    case "AGENT_HARNESS_CODEX":
      return "openai";
    case "AGENT_HARNESS_CLAUDE_CODE":
    case "AGENT_HARNESS_CURSOR":
    case "AGENT_HARNESS_SUPERPLANE":
      return null;
    default:
      return "claude";
  }
}

function isAgentProviderId(value: string | undefined): value is AgentProviderId {
  return AGENT_PROVIDER_IDS.some((id) => id === value);
}

export function restoreSavedAgentSelection(
  selections: IntegrationSelections,
  instances: OrganizationsIntegration[],
  agentIntegrationId: string | undefined,
): IntegrationSelections | null {
  const savedId = agentIntegrationId?.trim() ?? "";
  if (!savedId) return null;
  const instance = instances.find((item) => item.metadata?.id === savedId);
  const provider = instance?.metadata?.integrationName;
  if (!isAgentProviderId(provider) || instance?.status?.state !== "ready") return null;
  const selection = selectionFromInstance(instance);
  if (!selection?.ready) return null;
  const otherChoice = AGENT_PROVIDER_IDS.some((name) => {
    if (name === provider) return false;
    const current = selections[name];
    return Boolean(current && current.id !== savedId);
  });
  if (otherChoice) return null;
  const current = selections[provider];
  const misplaced = AGENT_PROVIDER_IDS.some((name) => name !== provider && selections[name]?.id === savedId);
  if (
    !misplaced &&
    current?.id === selection.id &&
    current.name === selection.name &&
    current.ready === selection.ready
  ) {
    return null;
  }
  const next = { ...selections };
  for (const name of AGENT_PROVIDER_IDS) {
    if (name !== provider && next[name]?.id === savedId) delete next[name];
  }
  next[provider] = selection;
  return next;
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
