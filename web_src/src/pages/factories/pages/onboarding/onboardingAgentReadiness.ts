import { pickHostedModel, pickModelMatching } from "@/lib/hostedLLMModels";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";

import type { IntegrationId } from "./onboardingFixtures";

export const AGENT_PROVIDER_IDS = ["claude", "openai", "openrouter"] as const;

export type AgentProviderId = (typeof AGENT_PROVIDER_IDS)[number];

export type HostedLLMProviderId = "anthropic" | "openai" | "openrouter";

export type OnboardingAgentHarness = "AGENT_HARNESS_CLAUDE_CODE" | "AGENT_HARNESS_CODEX";

export type OnboardingAgentPlan = {
  providerId: AgentProviderId;
  component: "runnerClaudeCode" | "runnerCodex" | "runnerOpenRouter";
  credentialsSource: "integration" | "hosted";
  integrationName: AgentProviderId;
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
};

const AGENT_PROVIDER_SPECS: Record<AgentProviderId, AgentProviderSpec> = {
  claude: {
    component: "runnerClaudeCode",
    hostedProvider: "anthropic",
    harness: "AGENT_HARNESS_CLAUDE_CODE",
    defaultModel: "sonnet",
    defaultPlanningModel: "opus",
    planningModelHint: "opus",
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
    defaultPlanningModel: "anthropic/claude-opus-4-6",
    planningModelHint: "opus",
  },
};

// A hosted run only accepts a model id from the allowlist, so the planning
// model has to come from the same list as the standard model. An empty list
// means no allowlist applies, and the agent CLI resolves the alias itself.
function planningModelFor(spec: AgentProviderSpec, modelIds: string[], model: string): string {
  if (modelIds.length === 0) return spec.defaultPlanningModel;
  return pickModelMatching(modelIds, spec.planningModelHint) ?? model;
}

export function isAgentProviderConnected(connected: Set<IntegrationId>): boolean {
  return AGENT_PROVIDER_IDS.some((id) => connected.has(id));
}

export function isAgentStepReady(connected: Set<IntegrationId>, remainingCreditCents: number): boolean {
  return remainingCreditCents > 0 || isAgentProviderConnected(connected);
}

export function resolveOnboardingAgent(args: {
  connected: Set<IntegrationId>;
  remainingCreditCents: number;
  hostedModels: HostedModelsByProvider;
}): OnboardingAgentPlan | undefined {
  for (const providerId of AGENT_PROVIDER_IDS) {
    if (!args.connected.has(providerId)) continue;
    return planForConnectedProvider(providerId, args.hostedModels);
  }

  if (args.remainingCreditCents <= 0) return undefined;

  for (const providerId of AGENT_PROVIDER_IDS) {
    const spec = AGENT_PROVIDER_SPECS[providerId];
    const modelIds = args.hostedModels[spec.hostedProvider];
    const model = pickHostedModel(spec.hostedProvider, modelIds);
    if (!model) continue;
    return {
      providerId,
      component: spec.component,
      credentialsSource: "hosted",
      integrationName: providerId,
      harness: spec.harness,
      model,
      planningModel: planningModelFor(spec, modelIds, model),
    };
  }

  return undefined;
}

export function hostedModelsQueriesLoading(needHosted: boolean, queries: Array<{ isFetched: boolean }>): boolean {
  return needHosted && queries.some((query) => !query.isFetched);
}

/**
 * Hosted credentials answer the agent question for the whole organization, so
 * setup has nothing left to ask about the agent. A plan that still loads can
 * become a plan that needs a key, so setup waits for the hosted models.
 */
export function isHostedAgentReady(args: {
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
}): boolean {
  if (args.hostedModelsLoading) return false;
  return args.plan?.credentialsSource === "hosted";
}

export function firstWorkOrderAgentError(args: {
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
}): string | null {
  if (args.hostedModelsLoading) {
    return "Hosted models are still loading. Try again.";
  }
  if (args.plan) return null;
  if (args.remainingCreditCents <= 0) {
    return "Connect Anthropic, OpenAI, or OpenRouter, or use hosted credit.";
  }
  return "Ask an installation admin to enable SuperPlane-hosted models.";
}

export function shouldShowHostedCreditGrant(grantTotalCents: number): boolean {
  return grantTotalCents > 0;
}

export function hostedCreditGrantCopy(remainingCreditCents: number): string {
  if (remainingCreditCents > 0) {
    return `This organization has ${formatUsdCents(remainingCreditCents)} of hosted credit. You can continue without connecting your own keys.`;
  }
  return "Hosted credit is empty. Connect a provider to continue.";
}

function planForConnectedProvider(
  providerId: AgentProviderId,
  hostedModels: HostedModelsByProvider,
): OnboardingAgentPlan {
  const spec = AGENT_PROVIDER_SPECS[providerId];
  const modelIds = hostedModels[spec.hostedProvider];
  const model = pickHostedModel(spec.hostedProvider, modelIds) ?? spec.defaultModel;
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
