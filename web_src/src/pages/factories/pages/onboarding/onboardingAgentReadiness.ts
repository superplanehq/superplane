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

export function isAgentStepReady(
  connected: Set<IntegrationId>,
  remainingCreditCents: number,
  keyProvider: AgentProviderId | null,
): boolean {
  if (keyProvider) {
    return connected.has(keyProvider);
  }
  return remainingCreditCents > 0;
}

export function resolveOnboardingAgent(args: {
  connected: Set<IntegrationId>;
  remainingCreditCents: number;
  hostedModels: HostedModelsByProvider;
  keyProvider: AgentProviderId | null;
}): OnboardingAgentPlan | undefined {
  if (args.keyProvider) {
    if (!args.connected.has(args.keyProvider)) return undefined;
    return planForConnectedProvider(args.keyProvider, args.hostedModels);
  }

  if (args.remainingCreditCents <= 0) return undefined;

  return hostedSuperPlaneAgent(args.hostedModels);
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
    return "Use SuperPlane agent credit, or connect Anthropic, OpenAI, or OpenRouter.";
  }
  return "Ask an installation admin to enable SuperPlane-hosted models.";
}

export function shouldShowHostedCreditGrant(grantTotalCents: number): boolean {
  return grantTotalCents > 0;
}

export function hostedCreditGrantCopy(remainingCreditCents: number): string {
  if (remainingCreditCents > 0) {
    return `This organization has ${formatUsdCents(remainingCreditCents)} of SuperPlane agent credit. SuperPlane can run the agent without your own key.`;
  }
  return "SuperPlane agent credit is empty. Connect a provider to continue.";
}

function hostedSuperPlaneAgent(hostedModels: HostedModelsByProvider): OnboardingAgentPlan | undefined {
  const spec = AGENT_PROVIDER_SPECS.openrouter;
  const model = pickHostedModel(spec.hostedProvider, hostedModels[spec.hostedProvider]);
  if (!model) return undefined;
  return {
    providerId: "openrouter",
    component: spec.component,
    credentialsSource: "hosted",
    integrationName: "openrouter",
    harness: spec.harness,
    model,
  };
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
