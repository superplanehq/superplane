import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { hostedModelIds } from "@/lib/hostedLLMModels";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";

import {
  hasHostedDefaultModel,
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
  defaultHosted?: { provider?: string; model?: string; preferOwnKey?: boolean; usageLoading?: boolean },
) {
  const needHostedModels = isAgentProviderConnected(connected);
  const anthropic = useHostedLLMModels(organizationId, "anthropic", needHostedModels);
  const openai = useHostedLLMModels(organizationId, "openai", needHostedModels);
  const openrouter = useHostedLLMModels(organizationId, "openrouter", needHostedModels);
  return {
    remainingCreditCents,
    hostedModelsAvailable: hasHostedDefaultModel({
      defaultHostedProvider: defaultHosted?.provider,
      defaultHostedModel: defaultHosted?.model,
    }),
    hostedModelsAvailableLoading: defaultHosted?.usageLoading ?? false,
    hostedModelsLoading: hostedModelsQueriesLoading(needHostedModels, [anthropic, openai, openrouter]),
    plan: resolveOnboardingAgent({
      connected,
      hostedModels: {
        anthropic: hostedModelIds(anthropic.data?.models),
        openai: hostedModelIds(openai.data?.models),
        openrouter: hostedModelIds(openrouter.data?.models),
      },
      defaultHostedProvider: defaultHosted?.provider,
      defaultHostedModel: defaultHosted?.model,
      preferOwnKey: defaultHosted?.preferOwnKey,
    }),
  };
}

export function agentRewriteFromPlan(
  plan: OnboardingAgentPlan,
  selections: IntegrationSelections,
): FactoryAgentRewrite {
  if (plan.component === "runnerSuperPlane") {
    return {
      component: "runnerSuperPlane",
      model: "",
      planningModel: "",
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

export function useOnboardingAgentContext(
  organizationId: string,
  connected: Set<IntegrationId>,
  preferOwnKey: boolean,
) {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  return useOnboardingAgentPlan(organizationId, connected, remainingCreditCents, {
    provider: spend.data?.defaultHostedProvider,
    model: spend.data?.defaultHostedModel,
    preferOwnKey,
    usageLoading: spend.isLoading,
  });
}
