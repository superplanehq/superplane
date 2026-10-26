import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { usePermissions } from "@/contexts/usePermissions";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { useBYOKLLMModels, useSwitchFactoryModelSource } from "@/hooks/useLLMModelAllowlists";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import {
  type LLMModelsSwitchDialog,
  type LLMModelsSwitchProvider,
  type LLMModelsSwitchTarget,
  SwitchDialog,
} from "./FactorySettingsLLMModelsSwitchPreview";
import { ModelSourceBody } from "./llmModelSourceBody";
import { byokProviderProductName, ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";
import { workspaceModelSource } from "./workspaceModelSource";
import { useFactoryCodexConnection } from "../onboarding/useFactoryCodexConnection";

type BYOKQuery = ReturnType<typeof useBYOKLLMModels>;

const NAMED_BYOK_PROVIDERS = ["anthropic", "openai", "openrouter"] as const;

export function FactorySettingsOrganizationLLMModelsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("org", "update") && !permissionsLoading;
  const integrationsHref = useIntegrationsBasePath(organizationId);
  const { allowed: byokAllowed, customProviderAllowed: customProviderEnabled } = useBYOKAccess(organizationId);
  const providers: LLMModelsSwitchProvider[] = !byokAllowed
    ? []
    : customProviderEnabled
      ? [...NAMED_BYOK_PROVIDERS, "custom"]
      : [...NAMED_BYOK_PROVIDERS];

  const anthropic = useBYOKLLMModels(organizationId, "anthropic", true);
  const openai = useBYOKLLMModels(organizationId, "openai", true);
  const openrouter = useBYOKLLMModels(organizationId, "openrouter", true);
  const custom = useBYOKLLMModels(organizationId, "custom", customProviderEnabled);
  const byokQueries: Record<string, BYOKQuery> = { anthropic, openai, openrouter, custom };
  const connected = providers.filter((provider) => {
    const query = byokQueries[provider];
    return query.data?.connected || (query.isError && !query.isLoading);
  });
  const byokLoading = providers.some((provider) => byokQueries[provider].isLoading);

  const subscription = useCodexSubscriptionCatalog(
    organizationId,
    factoryId,
    factory.onboarding?.codexSubscription === true,
  );

  const source = workspaceModelSource(factory.onboarding?.agentHarness, connected.length > 0);
  const currentProvider = subscription.selected
    ? null
    : resolveCurrentProvider(factory.onboarding, connected, byokQueries);
  const [dialog, setDialog] = useState<LLMModelsSwitchDialog>(null);
  const [switchedNotice, setSwitchedNotice] = useState<string | null>(null);
  const switchSource = useSwitchFactoryModelSource(organizationId, factoryId);

  usePageTitle([COPY.pageTitle, "Settings", factory.name ?? "Workspace"]);

  const saveSwitch = async (
    target: LLMModelsSwitchTarget,
    connection?: { apiKey?: string; baseUrl?: string; apiType?: string },
  ) => {
    try {
      await switchSource.mutateAsync({
        source: target,
        apiKey: connection?.apiKey,
        ...(target === "custom" ? { baseUrl: connection?.baseUrl, apiType: connection?.apiType } : {}),
      });
      setDialog(null);
      setSwitchedNotice(
        target === "hosted"
          ? "Automations in this workspace now use the SuperPlane agent."
          : target === "custom"
            ? "Automations in this workspace now use your custom provider."
            : `Automations in this workspace now use the ${byokProviderProductName(target)} agent.`,
      );
      showSuccessToast("Model source saved.");
    } catch (switchError) {
      showErrorToast(getApiErrorMessage(switchError, "Unable to switch the model source."));
    }
  };

  return (
    <FactorySettingsPageFrame title={COPY.pageTitle} subtitle={COPY.pageSubtitle}>
      <div data-testid="factory-settings-llm-models" className="flex flex-col gap-5">
        <FactorySettingsCard
          title={COPY.modelsTitle}
          data-testid="llm-models-list"
          action={
            <Badge variant="outline" data-testid="llm-models-source-badge">
              {modelSourceBadge(source, subscription.selected)}
            </Badge>
          }
        >
          <ModelSourceBody
            source={source}
            switchedNotice={switchedNotice}
            providers={providers}
            currentProvider={currentProvider}
            organizationId={organizationId}
            byokQueries={byokQueries}
            byokLoading={byokLoading}
            canUpdate={canUpdate}
            integrationsHref={integrationsHref}
            {...subscription.body}
            onChoose={(target) => setDialog({ target, step: "warn" })}
          />
        </FactorySettingsCard>
      </div>
      <SwitchDialog
        source={source === "hosted" ? "hosted" : (currentProvider ?? "anthropic")}
        dialog={dialog}
        onCancel={() => setDialog(null)}
        onContinueToKey={() => {
          if (!dialog || dialog.target === "hosted") {
            return;
          }
          if (connected.includes(dialog.target)) {
            void saveSwitch(dialog.target);
            return;
          }
          setDialog({ target: dialog.target, step: "key" });
        }}
        onBackToWarning={() => {
          if (dialog) {
            setDialog({ target: dialog.target, step: "warn" });
          }
        }}
        onSaveKey={(apiKey) => {
          if (dialog && dialog.target !== "hosted") {
            void saveSwitch(dialog.target, { apiKey });
          }
        }}
        onSaveCustom={(connection) => {
          void saveSwitch("custom", connection);
        }}
        onSwitchToHosted={() => {
          void saveSwitch("hosted");
        }}
      />
    </FactorySettingsPageFrame>
  );
}

function useCodexSubscriptionCatalog(organizationId: string, factoryId: string, selected: boolean) {
  const codex = useFactoryCodexConnection(organizationId, factoryId, selected);
  const connection = codex.query.data;
  const models = connection?.availableModels ?? [];
  const loading =
    selected && (codex.query.isLoading || connection?.reservationHeld === true || codex.refreshCatalog.isPending);
  const reconnectRequired = connection?.status === "reconnect_required" || connection?.status === "disconnected";

  return {
    selected,
    body: {
      subscriptionModels: selected ? models : undefined,
      subscriptionLoading: loading,
      subscriptionReconnectRequired: reconnectRequired,
    },
  };
}

function modelSourceBadge(source: "hosted" | "own-key", usesCodexSubscription: boolean): string {
  if (usesCodexSubscription) return COPY.subscriptionBadge;
  return source === "hosted" ? COPY.hostedBadge : COPY.ownKeyBadge;
}

function resolveCurrentProvider(
  onboarding: { agentHarness?: string; agentIntegrationId?: string } | undefined,
  connected: string[],
  queries: Record<string, BYOKQuery>,
): LLMModelsSwitchProvider | null {
  const integrationId = onboarding?.agentIntegrationId;
  if (integrationId) {
    const match = connected.find((provider) => queries[provider]?.data?.integrationId === integrationId);
    if (isSwitchProvider(match)) {
      return match;
    }
  }
  if (onboarding?.agentHarness === "AGENT_HARNESS_CODEX" && connected.includes("openai")) {
    return "openai";
  }
  if (onboarding?.agentHarness === "AGENT_HARNESS_CLAUDE_CODE") {
    if (connected.includes("anthropic")) return "anthropic";
    if (connected.includes("openrouter")) return "openrouter";
  }
  const first = connected.find(isSwitchProvider);
  return first ?? null;
}

function isSwitchProvider(value: string | undefined): value is LLMModelsSwitchProvider {
  return value === "anthropic" || value === "openai" || value === "openrouter" || value === "custom";
}
