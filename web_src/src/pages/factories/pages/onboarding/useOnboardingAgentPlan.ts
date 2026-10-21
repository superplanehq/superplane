import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { hostedModelIds } from "@/lib/hostedLLMModels";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";

import {
  AGENT_PROVIDER_IDS,
  hasHostedDefaultModel,
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
  selectedModel?: string | null;
};

export function useOnboardingAgentPlan(
  organizationId: string,
  connected: Set<IntegrationId>,
  remainingCreditCents: number,
  defaultHosted: HostedAgentDefaults = {},
) {
  const queries = {
    claude: useBYOKLLMModels(organizationId, "anthropic", connected.has("claude")),
    openai: useBYOKLLMModels(organizationId, "openai", connected.has("openai")),
    openrouter: useBYOKLLMModels(organizationId, "openrouter", connected.has("openrouter")),
    customLlm: useBYOKLLMModels(
      organizationId,
      "custom",
      Boolean(defaultHosted.customProvider) && connected.has("customLlm"),
    ),
  };
  const connectedForPlan = new Set(connected);
  if (!defaultHosted.customProvider) connectedForPlan.delete("customLlm");
  const plan = resolveOnboardingAgent({
    connected: connectedForPlan,
    hostedModels: {
      anthropic: byokModelIds(queries.claude.data),
      openai: byokModelIds(queries.openai.data),
      openrouter: byokModelIds(queries.openrouter.data),
    },
    customModels: byokModelIds(queries.customLlm.data),
    defaultHostedProvider: defaultHosted.provider,
    defaultHostedModel: defaultHosted.model,
    preferOwnKey: defaultHosted.preferOwnKey,
  });
  const providerId = plan?.providerId ?? AGENT_PROVIDER_IDS.find((id) => connectedForPlan.has(id));
  const query = plan?.credentialsSource !== "hosted" && providerId ? queries[providerId] : undefined;
  return {
    remainingCreditCents,
    hostedModelsAvailable: hasHostedDefaultModel({
      defaultHostedProvider: defaultHosted.provider,
      defaultHostedModel: defaultHosted.model,
    }),
    hostedModelsAvailableLoading: defaultHosted.usageLoading ?? false,
    hostedModelsLoading: Boolean(query && !query.isFetched),
    ...onboardingModelChoice(plan, query, defaultHosted.selectedModel),
  };
}

function onboardingModelChoice(
  plan: OnboardingAgentPlan | undefined,
  query: ReturnType<typeof useBYOKLLMModels> | undefined,
  selectedModel: string | null | undefined,
) {
  if (!query) return { plan, modelPicker: undefined };
  const models = agentUsableModelIds(byokModelIds(query.data));
  const defaultModel = plan && models.includes(plan.model) ? plan.model : (models[0] ?? "");
  const selectedIsUnusable = Boolean(selectedModel) && agentUsableModelIds([selectedModel ?? ""]).length === 0;
  const model = selectedIsUnusable ? defaultModel : (selectedModel ?? defaultModel);
  const ready = model !== "" && models.includes(model) && query.isFetched && !query.isError;
  return {
    modelPicker: {
      models,
      value: ready ? model : "",
      loading: !query.isFetched,
      error: query.isError,
      retry: () => {
        void query.refetch();
      },
    },
    plan: ready && plan ? { ...plan, model, planningModel: model } : undefined,
  };
}

const NON_AGENT_MODEL_SEGMENTS = new Set([
  "audio",
  "babbage",
  "dall",
  "davinci",
  "embedding",
  "embeddings",
  "image",
  "moderation",
  "moderations",
  "realtime",
  "sora",
  "transcribe",
  "transcription",
  "tts",
  "whisper",
]);

export function agentUsableModelIds(ids: string[]): string[] {
  return ids.filter((id) => {
    const segments = id
      .trim()
      .toLowerCase()
      .split(/[^a-z0-9]+/)
      .filter(Boolean);
    return segments.length > 0 && !segments.some((segment) => NON_AGENT_MODEL_SEGMENTS.has(segment));
  });
}

/** Keep organization selections within the models available to this key. */
export function byokModelIds(
  data: { selected?: { id?: string | null }[]; candidates?: { id?: string | null }[] } | undefined,
): string[] {
  const selected = hostedModelIds(data?.selected);
  const candidates = hostedModelIds(data?.candidates);
  return selected.length > 0 ? selected.filter((id) => candidates.includes(id)) : candidates;
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
  if (plan.credentialsSource === "subscription") {
    return {
      component: "runnerCodex",
      model: "",
      planningModel: "",
      credentials: { source: "subscription" },
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
  selectedModel?: string | null,
) {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  return useOnboardingAgentPlan(organizationId, connected, remainingCreditCents, {
    provider: spend.data?.defaultHostedProvider,
    model: spend.data?.defaultHostedModel,
    preferOwnKey,
    customProvider,
    selectedModel,
    usageLoading: spend.isLoading,
  });
}
