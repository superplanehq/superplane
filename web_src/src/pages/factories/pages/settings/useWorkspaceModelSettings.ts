import { useState } from "react";

import type { FactoriesFactory } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";

import type { LLMModelsSwitchProvider, LLMModelsSwitchTarget } from "./FactorySettingsLLMModelsSwitchPreview";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { workspaceModelSource } from "./workspaceModelSource";
import { useModelSourceSwitch } from "./useModelSourceSwitch";

type BYOKQuery = ReturnType<typeof useBYOKLLMModels>;
type SwitchTarget = LLMModelsSwitchTarget | "subscription";
const NAMED_BYOK_PROVIDERS: LLMModelsSwitchProvider[] = ["anthropic", "openai", "openrouter"];

function useModelProviderConnections(organizationId: string, factoryId: string, canRead: boolean) {
  const access = useBYOKAccess(organizationId);
  const providers = byokProviders(access.allowed, access.customProviderAllowed);
  const anthropic = useBYOKLLMModels(organizationId, "anthropic", canRead, factoryId);
  const openai = useBYOKLLMModels(organizationId, "openai", canRead, factoryId);
  const openrouter = useBYOKLLMModels(organizationId, "openrouter", canRead, factoryId);
  const custom = useBYOKLLMModels(organizationId, "custom", canRead && access.customProviderAllowed, factoryId);
  const byokQueries: Record<string, BYOKQuery> = { anthropic, openai, openrouter, custom };
  const connected = providers.filter(
    (provider) => byokQueries[provider].data?.connected || byokQueries[provider].isError,
  );
  return { providers, byokQueries, connected };
}

function byokProviders(allowed: boolean, customAllowed: boolean) {
  if (!allowed) return [];
  return customAllowed ? [...NAMED_BYOK_PROVIDERS, "custom" as const] : NAMED_BYOK_PROVIDERS;
}

function sourceForTarget(target: SwitchTarget) {
  if (target === "hosted" || target === "subscription") return target;
  return "own-key" as const;
}

function useCommittedModelSource(factory: FactoriesFactory, providers: ReturnType<typeof useModelProviderConnections>) {
  const baseline = JSON.stringify(factory.onboarding);
  const [committed, setCommitted] = useState<{ baseline: string; target: SwitchTarget } | null>(null);
  const target = committed?.baseline === baseline ? committed.target : null;
  const subscriptionSelected = target ? target === "subscription" : factory.onboarding?.codexSubscription === true;
  const source = target
    ? sourceForTarget(target)
    : workspaceModelSource(factory.onboarding?.agentHarness, providers.connected.length > 0, subscriptionSelected);
  const currentProvider =
    target && sourceForTarget(target) === "own-key"
      ? (target as LLMModelsSwitchProvider)
      : resolveCurrentProvider(factory.onboarding, providers.connected, providers.byokQueries);
  return {
    source,
    currentProvider,
    subscriptionSelected,
    onSaved: (next: SwitchTarget) => setCommitted({ baseline, target: next }),
  };
}

export function useWorkspaceModelSettings() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const permissions = usePermissions();
  const canRead = permissions.canAct("factories", "read") && !permissions.isLoading;
  const canUpdate = permissions.canAct("factories", "update") && !permissions.isLoading;
  const canCreateIntegration = permissions.canAct("integrations", "create") && !permissions.isLoading;
  const providers = useModelProviderConnections(organizationId, factoryId, canRead);
  const selection = useCommittedModelSource(factory, providers);
  const subscription = useCodexSubscriptionAccess();
  const switcher = useModelSourceSwitch(
    organizationId,
    factoryId,
    selection.onSaved,
    factory.onboarding?.codexDefaultOwnerId,
  );
  const integrationsHref = useIntegrationsBasePath(organizationId);
  const chooseSource = (target: string) => {
    if (target === selection.source || !canUpdate || switcher.saving) return;
    if (target === "subscription") {
      switcher.setSubscriptionDialog("warn");
      return;
    }
    if (target === "hosted") {
      switcher.setDialog({ target: "hosted", step: "warn" });
      return;
    }
    const provider =
      selection.currentProvider ??
      providers.connected[0] ??
      (canCreateIntegration ? providers.providers[0] : undefined);
    if (provider) switcher.setDialog({ target: provider, step: "warn" });
  };
  return {
    organizationId,
    factoryId,
    factory,
    permissionsLoading: permissions.isLoading,
    canRead,
    canUpdate,
    canCreateIntegration,
    ...providers,
    ...selection,
    subscription,
    switcher,
    integrationsHref,
    chooseSource,
  };
}

function useCodexSubscriptionAccess() {
  const { canAct } = usePermissions();
  return { offered: canAct("factories", "read") };
}

function resolveCurrentProvider(
  onboarding: FactoriesFactory["onboarding"],
  connected: LLMModelsSwitchProvider[],
  queries: Record<string, BYOKQuery>,
): LLMModelsSwitchProvider | null {
  const integrationId = onboarding?.agentIntegrationId;
  const match = connected.find((provider) => queries[provider]?.data?.integrationId === integrationId);
  if (integrationId && match) return match;
  if (onboarding?.agentHarness === "AGENT_HARNESS_CODEX" && connected.includes("openai")) return "openai";
  if (onboarding?.agentHarness === "AGENT_HARNESS_CLAUDE_CODE") {
    if (connected.includes("anthropic")) return "anthropic";
    if (connected.includes("openrouter")) return "openrouter";
  }
  return connected[0] ?? null;
}
