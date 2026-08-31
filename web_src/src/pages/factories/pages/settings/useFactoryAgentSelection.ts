import { useEffect, useMemo, useState } from "react";

export type AgentProvider = "anthropic" | "openai" | "openrouter";
export type CredentialSource = "hosted" | "integration";

export type ProviderOption = {
  provider: AgentProvider;
  integrationName: "claude" | "openai" | "openrouter";
  runner: string;
  codingModel: string;
  planningModel: string;
  codingHint: string;
  planningHint: string;
};

export const factoryAgentProviders: ProviderOption[] = [
  {
    provider: "anthropic",
    integrationName: "claude",
    runner: "Claude Code",
    codingModel: "sonnet",
    planningModel: "opus",
    codingHint: "sonnet",
    planningHint: "opus",
  },
  {
    provider: "openai",
    integrationName: "openai",
    runner: "Codex",
    codingModel: "gpt-5",
    planningModel: "gpt-5",
    codingHint: "gpt-5",
    planningHint: "gpt-5",
  },
  {
    provider: "openrouter",
    integrationName: "openrouter",
    runner: "OpenRouter",
    codingModel: "anthropic/claude-sonnet-4-6",
    planningModel: "anthropic/claude-opus-4-6",
    codingHint: "sonnet",
    planningHint: "opus",
  },
];

type AgentIntegration = {
  metadata?: { id?: string; integrationName?: string };
  status?: { state?: string };
};

type AgentOnboarding = {
  agentIntegrationId?: string;
  agentProvider?: string;
  agentHarness?: string;
};

export function useFactoryAgentSelection(onboarding: AgentOnboarding | undefined, integrations: AgentIntegration[]) {
  const [source, setSource] = useState<CredentialSource>(() => credentialSourceFor(onboarding?.agentIntegrationId));
  const [provider, setProvider] = useState<AgentProvider>(() => providerFor(onboarding));
  const [integrationId, setIntegrationId] = useState(onboarding?.agentIntegrationId ?? "");

  useEffect(() => {
    setSource(credentialSourceFor(onboarding?.agentIntegrationId));
    setProvider(providerFor(onboarding));
    setIntegrationId(onboarding?.agentIntegrationId ?? "");
  }, [onboarding]);

  const selectedProvider =
    factoryAgentProviders.find((option) => option.provider === provider) ?? factoryAgentProviders[0];
  const readyIntegrations = useMemo(
    () =>
      integrations.filter(
        (integration) =>
          integration.metadata?.integrationName === selectedProvider.integrationName &&
          integration.status?.state === "ready",
      ),
    [integrations, selectedProvider.integrationName],
  );
  useEffect(() => {
    if (source !== "integration" || integrationId || readyIntegrations.length === 0) return;
    setIntegrationId(readyIntegrations[0]?.metadata?.id ?? "");
  }, [integrationId, readyIntegrations, source]);
  useEffect(() => {
    if (onboarding?.agentProvider || !onboarding?.agentIntegrationId) return;
    const integration = integrations.find((item) => item.metadata?.id === onboarding.agentIntegrationId);
    const option = factoryAgentProviders.find(
      (item) => item.integrationName === integration?.metadata?.integrationName,
    );
    if (option) setProvider(option.provider);
  }, [integrations, onboarding?.agentIntegrationId, onboarding?.agentProvider]);

  return {
    source,
    setSource,
    provider,
    setProvider,
    integrationId,
    setIntegrationId,
    selectedProvider,
    readyIntegrations,
    dirty: isSelectionDirty({ source, provider, integrationId, onboarding }),
  };
}

export function providerFor(onboarding: AgentOnboarding | undefined): AgentProvider {
  switch (onboarding?.agentProvider) {
    case "AGENT_PROVIDER_OPENAI":
      return "openai";
    case "AGENT_PROVIDER_OPENROUTER":
      return "openrouter";
    default:
      return onboarding?.agentHarness === "AGENT_HARNESS_CODEX" ? "openai" : "anthropic";
  }
}

export function resolveAgentModels(
  provider: ProviderOption,
  source: CredentialSource,
  availableModels: Array<{ id?: string }>,
): { codingModel: string; planningModel: string } {
  if (source === "integration") {
    return { codingModel: provider.codingModel, planningModel: provider.planningModel };
  }
  const modelIDs = availableModels
    .map((model) => model.id ?? "")
    .filter(Boolean)
    .sort();
  if (modelIDs.length === 0) {
    return { codingModel: provider.codingModel, planningModel: provider.planningModel };
  }
  const codingModel =
    modelIDs.find((model) => model.toLowerCase().includes(provider.codingHint)) ?? modelIDs[0] ?? provider.codingModel;
  const planningModel = modelIDs.find((model) => model.toLowerCase().includes(provider.planningHint)) ?? codingModel;
  return { codingModel, planningModel };
}

export function countReadyAgentIntegrations(
  integrations: Array<{ metadata?: { integrationName?: string }; status?: { state?: string } }>,
  name: string,
) {
  return integrations.filter(
    (integration) => integration.metadata?.integrationName === name && integration.status?.state === "ready",
  ).length;
}

export function agentProviderToApi(
  provider: AgentProvider,
): "AGENT_PROVIDER_ANTHROPIC" | "AGENT_PROVIDER_OPENAI" | "AGENT_PROVIDER_OPENROUTER" {
  switch (provider) {
    case "anthropic":
      return "AGENT_PROVIDER_ANTHROPIC";
    case "openai":
      return "AGENT_PROVIDER_OPENAI";
    case "openrouter":
      return "AGENT_PROVIDER_OPENROUTER";
  }
}

function credentialSourceFor(integrationId: string | undefined): CredentialSource {
  return integrationId ? "integration" : "hosted";
}

function isSelectionDirty({
  source,
  provider,
  integrationId,
  onboarding,
}: {
  source: CredentialSource;
  provider: AgentProvider;
  integrationId: string;
  onboarding: AgentOnboarding | undefined;
}): boolean {
  if (source !== credentialSourceFor(onboarding?.agentIntegrationId)) return true;
  if (provider !== providerFor(onboarding)) return true;
  if (source !== "integration") return false;
  return integrationId !== (onboarding?.agentIntegrationId ?? "");
}
