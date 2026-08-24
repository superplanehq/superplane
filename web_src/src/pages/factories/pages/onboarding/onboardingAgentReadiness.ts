import { pickHostedModel } from "@/lib/hostedLLMModels";
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
};

export type HostedModelsByProvider = Record<HostedLLMProviderId, string[]>;

type AgentProviderSpec = {
  component: OnboardingAgentPlan["component"];
  hostedProvider: HostedLLMProviderId;
  harness: OnboardingAgentHarness;
  defaultModel: string;
};

const AGENT_PROVIDER_SPECS: Record<AgentProviderId, AgentProviderSpec> = {
  claude: {
    component: "runnerClaudeCode",
    hostedProvider: "anthropic",
    harness: "AGENT_HARNESS_CLAUDE_CODE",
    defaultModel: "sonnet",
  },
  openai: {
    component: "runnerCodex",
    hostedProvider: "openai",
    harness: "AGENT_HARNESS_CODEX",
    defaultModel: "gpt-5",
  },
  openrouter: {
    component: "runnerOpenRouter",
    hostedProvider: "openrouter",
    harness: "AGENT_HARNESS_CLAUDE_CODE",
    defaultModel: "anthropic/claude-sonnet-4-6",
  },
};

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
    const model = pickHostedModel(spec.hostedProvider, args.hostedModels[spec.hostedProvider]);
    if (!model) continue;
    return {
      providerId,
      component: spec.component,
      credentialsSource: "hosted",
      integrationName: providerId,
      harness: spec.harness,
      model,
    };
  }

  return undefined;
}

export function hostedModelsQueriesLoading(needHosted: boolean, queries: Array<{ isFetched: boolean }>): boolean {
  return needHosted && queries.some((query) => !query.isFetched);
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
  return {
    providerId,
    component: spec.component,
    credentialsSource: "integration",
    integrationName: providerId,
    harness: spec.harness,
    model: pickHostedModel(spec.hostedProvider, hostedModels[spec.hostedProvider]) ?? spec.defaultModel,
  };
}
