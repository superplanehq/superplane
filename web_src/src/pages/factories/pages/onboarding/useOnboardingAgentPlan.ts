import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { hostedModelIds } from "@/lib/hostedLLMModels";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";

import {
  hostedModelsQueriesLoading,
  isAgentProviderConnected,
  resolveOnboardingAgent,
  type OnboardingAgentPlan,
} from "./onboardingAgentReadiness";
import type { IntegrationId } from "./onboardingFixtures";

export function useOnboardingAgentPlan(
  organizationId: string,
  connected: Set<IntegrationId>,
  remainingCreditCents: number,
  defaultHosted?: { provider?: string; model?: string },
) {
  const needHostedModels = isAgentProviderConnected(connected);
  const anthropic = useHostedLLMModels(organizationId, "anthropic", needHostedModels);
  const openai = useHostedLLMModels(organizationId, "openai", needHostedModels);
  const openrouter = useHostedLLMModels(organizationId, "openrouter", needHostedModels);
  return {
    remainingCreditCents,
    hostedModelsLoading: hostedModelsQueriesLoading(needHostedModels, [anthropic, openai, openrouter]),
    plan: resolveOnboardingAgent({
      connected,
      remainingCreditCents,
      hostedModels: {
        anthropic: hostedModelIds(anthropic.data?.models),
        openai: hostedModelIds(openai.data?.models),
        openrouter: hostedModelIds(openrouter.data?.models),
      },
      defaultHostedProvider: defaultHosted?.provider,
      defaultHostedModel: defaultHosted?.model,
    }),
  };
}

export function agentRewriteFromPlan(
  plan: OnboardingAgentPlan,
  selections: IntegrationSelections,
): FactoryAgentRewrite {
  if (plan.component === "runnerSuperPlane" || plan.credentialsSource === "hosted") {
    return {
      component: "runnerSuperPlane",
      model: "",
      planningModel: "",
      credentials: { source: "hosted" },
    };
  }
  const integrationName = plan.integrationName;
  if (!integrationName) {
    throw new Error("Agent integration is missing");
  }
  return {
    component: plan.component,
    model: plan.model,
    planningModel: plan.planningModel,
    credentials: {
      source: "integration",
      name: selections[integrationName]?.name ?? integrationName,
    },
  };
}
