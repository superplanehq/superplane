import type { FactoriesFactory, FactoriesUpdateFactoryOnboardingBody, OrganizationsIntegration } from "@/api-client";
import {
  selectionFromInstance,
  type IntegrationSelection,
  type IntegrationSelections,
} from "@/pages/home/homeIntegrationStatus";

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

type SavedAgentSelection = {
  id: string;
  provider: AgentProviderId;
  selection: IntegrationSelection;
};

function readySavedAgent(
  instances: OrganizationsIntegration[],
  agentIntegrationId: string | undefined,
): SavedAgentSelection | null {
  const savedId = agentIntegrationId?.trim() ?? "";
  if (!savedId) return null;
  const instance = instances.find((item) => item.metadata?.id === savedId);
  if (!instance || instance.status?.state !== "ready") return null;
  const provider = instance.metadata?.integrationName;
  if (!isAgentProviderId(provider)) return null;
  const selection = selectionFromInstance(instance);
  if (!selection?.ready) return null;
  return { id: savedId, provider, selection };
}

function choseAnotherAgent(selections: IntegrationSelections, saved: SavedAgentSelection): boolean {
  return AGENT_PROVIDER_IDS.some((name) => {
    if (name === saved.provider) return false;
    const current = selections[name];
    if (!current) return false;
    return current.id !== saved.id;
  });
}

function savedAgentAlreadyApplied(selections: IntegrationSelections, saved: SavedAgentSelection): boolean {
  const current = selections[saved.provider];
  if (!current) return false;
  if (current.id !== saved.selection.id) return false;
  if (current.name !== saved.selection.name) return false;
  if (current.ready !== saved.selection.ready) return false;
  return !savedAgentIsMisplaced(selections, saved);
}

function savedAgentIsMisplaced(selections: IntegrationSelections, saved: SavedAgentSelection): boolean {
  return AGENT_PROVIDER_IDS.some((name) => name !== saved.provider && selections[name]?.id === saved.id);
}

function withSavedAgent(selections: IntegrationSelections, saved: SavedAgentSelection): IntegrationSelections {
  const next = { ...selections };
  for (const name of AGENT_PROVIDER_IDS) {
    if (name === saved.provider) continue;
    if (next[name]?.id !== saved.id) continue;
    delete next[name];
  }
  next[saved.provider] = saved.selection;
  return next;
}

export function restoreSavedAgentSelection(
  selections: IntegrationSelections,
  instances: OrganizationsIntegration[],
  agentIntegrationId: string | undefined,
): IntegrationSelections | null {
  const saved = readySavedAgent(instances, agentIntegrationId);
  if (!saved) return null;
  if (choseAnotherAgent(selections, saved)) return null;
  if (savedAgentAlreadyApplied(selections, saved)) return null;
  return withSavedAgent(selections, saved);
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
