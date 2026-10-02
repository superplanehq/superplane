import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
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

type HostedAgentDefaults = {
  provider?: string;
  model?: string;
  preferOwnKey?: boolean;
  customProvider?: boolean;
  usageLoading?: boolean;
};

type ModelQuery = {
  data?: {
    models?: { id?: string | null }[];
    selected?: { id?: string | null }[];
    candidates?: { id?: string | null }[];
  };
  isFetched: boolean;
};

export function useOnboardingAgentPlan(
  organizationId: string,
  connected: Set<IntegrationId>,
  remainingCreditCents: number,
  defaultHosted?: HostedAgentDefaults,
) {
  const needHostedModels = isAgentProviderConnected(connected);
  const customConnected = customProviderConnected(connected, defaultHosted?.customProvider);
  // A Claude key picks from the models that key can use, not the hosted allowlist.
  const anthropic = useBYOKLLMModels(organizationId, "anthropic", needHostedModels);
  const openai = useHostedLLMModels(organizationId, "openai", needHostedModels);
  const openrouter = useHostedLLMModels(organizationId, "openrouter", needHostedModels);
  const custom = useBYOKLLMModels(organizationId, "custom", customConnected);
  return onboardingAgentPlan({
    connected,
    remainingCreditCents,
    defaultHosted,
    needHostedModels,
    customConnected,
    anthropic,
    openai,
    openrouter,
    custom,
  });
}

function customProviderConnected(connected: Set<IntegrationId>, enabled: boolean | undefined): boolean {
  return Boolean(enabled) && connected.has("customLlm");
}

function onboardingAgentPlan(args: {
  connected: Set<IntegrationId>;
  remainingCreditCents: number;
  defaultHosted?: HostedAgentDefaults;
  needHostedModels: boolean;
  customConnected: boolean;
  anthropic: ModelQuery;
  openai: ModelQuery;
  openrouter: ModelQuery;
  custom: ModelQuery;
}) {
  const connectedForPlan = new Set(args.connected);
  if (!args.customConnected) connectedForPlan.delete("customLlm");
  const modelQueries = [args.anthropic, args.openai, args.openrouter];
  if (args.customConnected) modelQueries.push(args.custom);
  return {
    remainingCreditCents: args.remainingCreditCents,
    hostedModelsAvailable: hasHostedDefaultModel({
      defaultHostedProvider: args.defaultHosted?.provider,
      defaultHostedModel: args.defaultHosted?.model,
    }),
    hostedModelsAvailableLoading: args.defaultHosted?.usageLoading ?? false,
    hostedModelsLoading: hostedModelsQueriesLoading(args.needHostedModels || args.customConnected, modelQueries),
    plan: resolveOnboardingAgent({
      connected: connectedForPlan,
      hostedModels: {
        anthropic: anthropicKeyModelIds(args.anthropic.data),
        openai: hostedModelIds(args.openai.data?.models),
        openrouter: hostedModelIds(args.openrouter.data?.models),
      },
      customModels: args.customConnected ? anthropicKeyModelIds(args.custom.data) : [],
      defaultHostedProvider: args.defaultHosted?.provider,
      defaultHostedModel: args.defaultHosted?.model,
      preferOwnKey: args.defaultHosted?.preferOwnKey,
    }),
  };
}

/** The organization's selected Claude models, or every model the key can use. */
export function anthropicKeyModelIds(
  data: { selected?: { id?: string | null }[]; candidates?: { id?: string | null }[] } | undefined,
): string[] {
  const selected = hostedModelIds(data?.selected);
  return selected.length > 0 ? selected : hostedModelIds(data?.candidates);
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
    ...(plan.llmProvider ? { llmProvider: plan.llmProvider } : {}),
  };
}

export function useOnboardingAgentContext(
  organizationId: string,
  connected: Set<IntegrationId>,
  preferOwnKey: boolean,
  customProvider = false,
) {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  return useOnboardingAgentPlan(organizationId, connected, remainingCreditCents, {
    provider: spend.data?.defaultHostedProvider,
    model: spend.data?.defaultHostedModel,
    preferOwnKey,
    customProvider,
    usageLoading: spend.isLoading,
  });
}
