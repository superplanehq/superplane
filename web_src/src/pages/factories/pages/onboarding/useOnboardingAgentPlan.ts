import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { hostedModelIds } from "@/lib/hostedLLMModels";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";

import {
  hostedModelsQueriesLoading,
  resolveOnboardingAgent,
  type AgentProviderId,
  type OnboardingAgentPlan,
} from "./onboardingAgentReadiness";
import type { IntegrationId } from "./onboardingFixtures";

export function useOnboardingAgentPlan(
  organizationId: string,
  connected: Set<IntegrationId>,
  remainingCreditCents: number,
  keyProvider: AgentProviderId | null,
) {
  const needHosted = remainingCreditCents > 0 && keyProvider === null;
  const openrouter = useHostedLLMModels(organizationId, "openrouter", needHosted);
  return {
    remainingCreditCents,
    hostedModelsLoading: hostedModelsQueriesLoading(needHosted, [openrouter]),
    plan: resolveOnboardingAgent({
      connected,
      remainingCreditCents,
      hostedModels: {
        anthropic: [],
        openai: [],
        openrouter: hostedModelIds(openrouter.data?.models),
      },
      keyProvider,
    }),
  };
}

export function agentRewriteFromPlan(
  plan: OnboardingAgentPlan,
  selections: IntegrationSelections,
): FactoryAgentRewrite {
  if (plan.credentialsSource === "hosted") {
    return { component: plan.component, model: plan.model, credentials: { source: "hosted" } };
  }
  return {
    component: plan.component,
    model: plan.model,
    credentials: {
      source: "integration",
      name: selections[plan.integrationName]?.name ?? plan.integrationName,
    },
  };
}
