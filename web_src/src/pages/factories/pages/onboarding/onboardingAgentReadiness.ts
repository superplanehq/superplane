import { newestClaudeModelInFamily, pickHostedModel, pickNewestModelMatching } from "@/lib/hostedLLMModels";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";

import type { IntegrationId } from "./onboardingFixtures";

export const AGENT_PROVIDER_IDS = ["claude", "openai", "openrouter"] as const;

export type AgentProviderId = (typeof AGENT_PROVIDER_IDS)[number];

export type HostedLLMProviderId = "anthropic" | "openai" | "openrouter";

export type OnboardingAgentHarness = "AGENT_HARNESS_CLAUDE_CODE" | "AGENT_HARNESS_CODEX" | "AGENT_HARNESS_SUPERPLANE";

export type OnboardingAgentPlan = {
  providerId?: AgentProviderId;
  component: "runnerClaudeCode" | "runnerCodex" | "runnerOpenRouter" | "runnerSuperPlane";
  credentialsSource: "integration" | "hosted";
  integrationName?: AgentProviderId;
  harness: OnboardingAgentHarness;
  model: string;
  /** Model for agents that weigh evidence rather than write code, such as planning. */
  planningModel: string;
};

export type HostedModelsByProvider = Record<HostedLLMProviderId, string[]>;

type AgentProviderSpec = {
  component: OnboardingAgentPlan["component"];
  hostedProvider: HostedLLMProviderId;
  harness: OnboardingAgentHarness;
  defaultModel: string;
  /** Default for planning-style agents when no allowlist is available. */
  defaultPlanningModel: string;
  /** Substring that finds the planning model on an allowlist. */
  planningModelHint: string;
  /** Claude family whose newest model on the key runs implementation. */
  modelFamily?: string;
  /** Claude family whose newest model on the key runs planning. */
  planningModelFamily?: string;
};

const AGENT_PROVIDER_SPECS: Record<AgentProviderId, AgentProviderSpec> = {
  claude: {
    component: "runnerClaudeCode",
    hostedProvider: "anthropic",
    harness: "AGENT_HARNESS_CLAUDE_CODE",
    defaultModel: "claude-sonnet-4-6",
    defaultPlanningModel: "claude-opus-5-5",
    planningModelHint: "opus",
    modelFamily: "sonnet",
    planningModelFamily: "opus",
  },
  openai: {
    component: "runnerCodex",
    hostedProvider: "openai",
    harness: "AGENT_HARNESS_CODEX",
    defaultModel: "gpt-5",
    defaultPlanningModel: "gpt-5",
    planningModelHint: "gpt-5",
  },
  openrouter: {
    component: "runnerOpenRouter",
    hostedProvider: "openrouter",
    harness: "AGENT_HARNESS_CLAUDE_CODE",
    defaultModel: "anthropic/claude-sonnet-4-6",
    defaultPlanningModel: "anthropic/claude-opus-5-5",
    planningModelHint: "opus",
  },
};

// A hosted run only accepts a model id from the allowlist, so the planning
// model has to come from the same list as the standard model. An empty list
// uses a versioned model id.
function planningModelFor(spec: AgentProviderSpec, modelIds: string[], model: string): string {
  if (modelIds.length === 0) return spec.defaultPlanningModel;
  if (spec.planningModelFamily) {
    return newestClaudeModelInFamily(modelIds, spec.planningModelFamily) ?? model;
  }
  return pickNewestModelMatching(modelIds, spec.planningModelHint) ?? model;
}

function implementationModelFor(spec: AgentProviderSpec, modelIds: string[]): string {
  const newest = spec.modelFamily ? newestClaudeModelInFamily(modelIds, spec.modelFamily) : undefined;
  return newest ?? pickHostedModel(spec.hostedProvider, modelIds) ?? spec.defaultModel;
}

export function isAgentProviderConnected(connected: Set<IntegrationId>): boolean {
  return AGENT_PROVIDER_IDS.some((id) => connected.has(id));
}

export function isAgentStepReady(connected: Set<IntegrationId>, remainingCreditCents: number): boolean {
  return remainingCreditCents > 0 || isAgentProviderConnected(connected);
}

export function resolveOnboardingAgent(args: {
  connected: Set<IntegrationId>;
  hostedModels: HostedModelsByProvider;
  defaultHostedProvider?: string;
  defaultHostedModel?: string;
  /** When set, a connected provider key is used before the hosted model. */
  preferOwnKey?: boolean;
}): OnboardingAgentPlan | undefined {
  if (args.preferOwnKey) {
    const ownKey = connectedProviderPlan(args);
    if (ownKey) return ownKey;
  }

  const hosted = hostedSuperPlanePlan(args);
  if (hosted) return hosted;
  if (args.preferOwnKey) return undefined;

  return connectedProviderPlan(args);
}

function connectedProviderPlan(args: {
  connected: Set<IntegrationId>;
  hostedModels: HostedModelsByProvider;
}): OnboardingAgentPlan | undefined {
  for (const providerId of AGENT_PROVIDER_IDS) {
    if (!args.connected.has(providerId)) continue;
    return planForConnectedProvider(providerId, args.hostedModels);
  }
  return undefined;
}

/** True when the installation sets a default SuperPlane-hosted model. */
export function hasHostedDefaultModel(args: { defaultHostedProvider?: string; defaultHostedModel?: string }): boolean {
  const defaultProvider = args.defaultHostedProvider?.trim() ?? "";
  const defaultModel = args.defaultHostedModel?.trim() ?? "";
  return Boolean(defaultProvider && defaultModel);
}

function hostedSuperPlanePlan(args: {
  defaultHostedProvider?: string;
  defaultHostedModel?: string;
}): OnboardingAgentPlan | undefined {
  if (!hasHostedDefaultModel(args)) return undefined;

  return {
    component: "runnerSuperPlane",
    credentialsSource: "hosted",
    harness: "AGENT_HARNESS_SUPERPLANE",
    model: "",
    planningModel: "",
  };
}

export function hostedModelsQueriesLoading(needHosted: boolean, queries: Array<{ isFetched: boolean }>): boolean {
  return needHosted && queries.some((query) => !query.isFetched);
}

/**
 * A hosted default can run the agent without a provider key. Billing controls
 * hosted runs after setup. Installations without a hosted default still use
 * the connection step.
 */
export function isHostedAgentReady(plan: OnboardingAgentPlan | undefined): boolean {
  return plan?.component === "runnerSuperPlane";
}

export type OnboardingAgentCredentialChoice = "own-key" | "hosted";

/**
 * Where the agent screen goes in the wizard:
 * - `show`: after the ticket screen, to connect a provider key.
 * - `first`: before the ticket screen, to choose a model source.
 * - `skip`: nowhere. Hosted models run the agent.
 * - `pending`: not known until the bring-your-own-key flag loads.
 */
export type OnboardingAgentGate = "show" | "first" | "skip" | "pending";

/**
 * Hosted models skip the agent screen. Only an organization with the
 * bring-your-own-key flag chooses its own provider key or SuperPlane-hosted
 * models, before it chooses the backlog. The gate waits until the flag has
 * loaded, so setup does not finish on the hosted path first.
 */
export function onboardingAgentGate(args: {
  hostedModelsAvailable: boolean;
  hostedModelsAvailableLoading: boolean;
  bringYourOwnKey: boolean;
  bringYourOwnKeyLoading: boolean;
}): OnboardingAgentGate {
  if (args.hostedModelsAvailableLoading || args.bringYourOwnKeyLoading) return "pending";
  if (!args.hostedModelsAvailable) return "show";
  if (args.bringYourOwnKey) return "first";
  return "skip";
}

/**
 * A model source choice must be made before setup finishes. Own-key setup
 * finishes only after a provider key is connected.
 */
export function agentFinishReady(args: {
  modelSourceChoice: boolean;
  credentialChoice: OnboardingAgentCredentialChoice | null;
  providerConnected: boolean;
  agentReady: boolean;
  hostedAgentReady: boolean;
}): boolean {
  if (args.modelSourceChoice && !args.credentialChoice) return false;
  if (args.credentialChoice === "own-key") return args.providerConnected;
  return args.agentReady || args.hostedAgentReady;
}

export function firstWorkOrderAgentError(args: {
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
}): string | null {
  if (args.plan?.component === "runnerSuperPlane") return null;
  if (args.hostedModelsLoading) {
    return "Hosted models are still loading. Try again.";
  }
  if (args.plan) return null;
  if (args.remainingCreditCents <= 0) {
    return "Connect Anthropic, OpenAI, or OpenRouter, or use hosted credit.";
  }
  return "Ask an installation admin to set a SuperPlane agent model.";
}

export function shouldShowHostedCreditGrant(grantTotalCents: number): boolean {
  return grantTotalCents > 0;
}

export function hostedCreditGrantCopy(remainingCreditCents: number): string {
  if (remainingCreditCents > 0) {
    return `This organization has ${formatUsdCents(remainingCreditCents)} of trial usage for machines and managed models. Subscribe to Business to keep hosted runs after the trial.`;
  }
  return "Trial credit is used up. Subscribe to Business or connect a provider to continue.";
}

function planForConnectedProvider(
  providerId: AgentProviderId,
  hostedModels: HostedModelsByProvider,
): OnboardingAgentPlan {
  const spec = AGENT_PROVIDER_SPECS[providerId];
  const modelIds = hostedModels[spec.hostedProvider];
  const model = implementationModelFor(spec, modelIds);
  return {
    providerId,
    component: spec.component,
    credentialsSource: "integration",
    integrationName: providerId,
    harness: spec.harness,
    model,
    planningModel: planningModelFor(spec, modelIds, model),
  };
}
