import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { hostedModelIds } from "@/lib/hostedLLMModels";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";

import {
  hostedModelsQueriesLoading,
  resolveOnboardingAgent,
  type OnboardingAgentPlan,
} from "./onboardingAgentReadiness";
import type { IntegrationId } from "./onboardingFixtures";

export function onboardingAgentProvider(providerId: OnboardingAgentPlan["providerId"]) {
  switch (providerId) {
    case "claude":
      return "AGENT_PROVIDER_ANTHROPIC" as const;
    case "openai":
      return "AGENT_PROVIDER_OPENAI" as const;
    case "openrouter":
      return "AGENT_PROVIDER_OPENROUTER" as const;
  }
}

export function useOnboardingAgentPlan(
  organizationId: string,
  connected: Set<IntegrationId>,
  remainingCreditCents: number,
) {
  const needHosted = remainingCreditCents > 0;
  const anthropic = useHostedLLMModels(organizationId, "anthropic", needHosted);
  const openai = useHostedLLMModels(organizationId, "openai", needHosted);
  const openrouter = useHostedLLMModels(organizationId, "openrouter", needHosted);
  return {
    remainingCreditCents,
    hostedModelsLoading: hostedModelsQueriesLoading(needHosted, [anthropic, openai, openrouter]),
    plan: resolveOnboardingAgent({
      connected,
      remainingCreditCents,
      hostedModels: {
        anthropic: hostedModelIds(anthropic.data?.models),
        openai: hostedModelIds(openai.data?.models),
        openrouter: hostedModelIds(openrouter.data?.models),
      },
    }),
  };
}

export function agentRewriteFromPlan(
  plan: OnboardingAgentPlan,
  selections: IntegrationSelections,
): FactoryAgentRewrite {
  if (plan.credentialsSource === "hosted") {
    return {
      component: plan.component,
      model: plan.model,
      planningModel: plan.planningModel,
      credentials: { source: "hosted" },
    };
  }
  return {
    component: plan.component,
    model: plan.model,
    planningModel: plan.planningModel,
    credentials: {
      source: "integration",
      name: selections[plan.integrationName]?.name ?? plan.integrationName,
    },
  };
}
