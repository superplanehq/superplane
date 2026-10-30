import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";
import { CodexConnectionSetupDialog } from "../../codex/CodexConnectionSetupDialog";
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
import { workspaceModelSource, type WorkspaceModelSource } from "./workspaceModelSource";
import { FactoryCodexSubscriptionConnection } from "./FactoryCodexSubscriptionConnection";
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

  const source = workspaceModelSource(factory.onboarding?.agentHarness, connected.length > 0, subscription.selected);
  const currentProvider = subscription.selected
    ? null
    : resolveCurrentProvider(factory.onboarding, connected, byokQueries);
  const switcher = useModelSourceSwitch(organizationId, factoryId);
  usePageTitle([COPY.pageTitle, "Settings", factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame title={COPY.pageTitle} subtitle={COPY.pageSubtitle}>
      <div data-testid="factory-settings-llm-models" className="flex flex-col gap-5">
        {subscription.selected ? <FactoryCodexSubscriptionConnection codex={subscription.codex} /> : null}
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
            switchedNotice={switcher.switchedNotice}
            providers={providers}
            currentProvider={currentProvider}
            organizationId={organizationId}
            byokQueries={byokQueries}
            byokLoading={byokLoading}
            canUpdate={canUpdate}
            integrationsHref={integrationsHref}
            {...subscription.body}
            onChoose={(target) => switcher.setDialog({ target, step: "warn" })}
          />
          {!subscription.selected && subscription.offered ? (
            <CodexSubscriptionChoice
              disabled={!canUpdate || switcher.saving}
              onChoose={() => switcher.setSubscriptionDialog("warn")}
            />
          ) : null}
        </FactorySettingsCard>
      </div>
      <ModelSourceSwitchDialogs
        organizationId={organizationId}
        factoryId={factoryId}
        source={source}
        currentProvider={currentProvider}
        connected={connected}
        switcher={switcher}
      />
    </FactorySettingsPageFrame>
  );
}

function useModelSourceSwitch(organizationId: string, factoryId: string) {
  const [subscriptionDialog, setSubscriptionDialog] = useState<"warn" | "connect" | null>(null);
  const [dialog, setDialog] = useState<LLMModelsSwitchDialog>(null);
  const [switchedNotice, setSwitchedNotice] = useState<string | null>(null);
  const switchSource = useSwitchFactoryModelSource(organizationId, factoryId);

  const saveSwitch = async (
    target: LLMModelsSwitchTarget | "subscription",
    connection?: { apiKey?: string; baseUrl?: string; apiType?: string },
  ) => {
    try {
      await switchSource.mutateAsync({
        source: target,
        apiKey: connection?.apiKey,
        ...(target === "custom" ? { baseUrl: connection?.baseUrl, apiType: connection?.apiType } : {}),
      });
      setDialog(null);
      setSubscriptionDialog(null);
      setSwitchedNotice(
        target === "subscription"
          ? "Automations in this workspace now use Codex subscriptions."
          : target === "hosted"
            ? "Automations in this workspace now use the SuperPlane agent."
            : target === "custom"
              ? "Automations in this workspace now use your custom provider."
              : `Automations in this workspace now use the ${byokProviderProductName(target)} agent.`,
      );
      showSuccessToast("Model source saved.");
      return true;
    } catch (switchError) {
      showErrorToast(getApiErrorMessage(switchError, "Unable to switch the model source."));
      return false;
    }
  };

  return {
    dialog,
    setDialog,
    subscriptionDialog,
    setSubscriptionDialog,
    switchedNotice,
    saveSwitch,
    saving: switchSource.isPending,
  };
}

function CodexSubscriptionChoice({ disabled, onChoose }: { disabled: boolean; onChoose: () => void }) {
  return (
    <div className="mt-5 flex items-center justify-between gap-3 border-t border-border pt-4">
      <div>
        <p className="text-[13px] font-medium">Codex subscription</p>
        <p className="mt-1 text-xs text-muted-foreground">Each task owner uses their personal Codex connection.</p>
      </div>
      <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={onChoose}>
        Use Codex subscription
      </Button>
    </div>
  );
}

function ModelSourceSwitchDialogs({
  organizationId,
  factoryId,
  source,
  currentProvider,
  connected,
  switcher,
}: {
  organizationId: string;
  factoryId: string;
  source: WorkspaceModelSource;
  currentProvider: LLMModelsSwitchProvider | null;
  connected: LLMModelsSwitchProvider[];
  switcher: ReturnType<typeof useModelSourceSwitch>;
}) {
  return (
    <>
      <Dialog
        open={switcher.subscriptionDialog === "warn"}
        onOpenChange={(open) => {
          if (!open) switcher.setSubscriptionDialog(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Switch to Codex subscriptions?</DialogTitle>
            <DialogDescription>
              Connect your Codex account before this workspace switches. Intakes without a connected default owner will
              use your account.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => switcher.setSubscriptionDialog(null)}>
              Cancel
            </Button>
            <Button type="button" onClick={() => switcher.setSubscriptionDialog("connect")}>
              Connect Codex
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <CodexConnectionSetupDialog
        organizationId={organizationId}
        factoryId={factoryId}
        open={switcher.subscriptionDialog === "connect"}
        onOpenChange={(open) => {
          if (!open) switcher.setSubscriptionDialog(null);
        }}
        onConnected={async () => {
          if (!(await switcher.saveSwitch("subscription")))
            throw new Error("Could not switch the model source. Try again.");
        }}
      />
      <SwitchDialog
        source={source === "hosted" || source === "subscription" ? source : (currentProvider ?? "anthropic")}
        dialog={switcher.dialog}
        onCancel={() => switcher.setDialog(null)}
        onContinueToKey={() => {
          if (!switcher.dialog || switcher.dialog.target === "hosted") {
            return;
          }
          if (connected.includes(switcher.dialog.target)) {
            void switcher.saveSwitch(switcher.dialog.target);
            return;
          }
          switcher.setDialog({ target: switcher.dialog.target, step: "key" });
        }}
        onBackToWarning={() => {
          if (switcher.dialog) {
            switcher.setDialog({ target: switcher.dialog.target, step: "warn" });
          }
        }}
        onSaveKey={(apiKey) => {
          if (switcher.dialog && switcher.dialog.target !== "hosted") {
            void switcher.saveSwitch(switcher.dialog.target, { apiKey });
          }
        }}
        onSaveCustom={(connection) => {
          void switcher.saveSwitch("custom", connection);
        }}
        onSwitchToHosted={() => {
          void switcher.saveSwitch("hosted");
        }}
      />
    </>
  );
}

function useCodexSubscriptionCatalog(organizationId: string, factoryId: string, selected: boolean) {
  const features = useExperimentalFeature(organizationId);
  const { canAct } = usePermissions();
  const enabled = [!features.isLoading, features.has(FEATURE_CODEX_SUBSCRIPTIONS), canAct("factories", "read")].every(
    Boolean,
  );
  const codex = useFactoryCodexConnection(organizationId, factoryId, selected || enabled, selected);
  const connection = codex.query.data;
  const models = connection?.availableModels ?? [];
  const loading =
    selected && (codex.query.isLoading || connection?.reservationHeld === true || codex.refreshCatalog.isPending);
  const reconnectRequired = connection?.status === "reconnect_required" || connection?.status === "disconnected";

  return {
    selected,
    offered: enabled && connection?.eligible === true,
    codex,
    body: {
      subscriptionModels: selected ? models : undefined,
      subscriptionLoading: loading,
      subscriptionReconnectRequired: reconnectRequired,
    },
  };
}

function modelSourceBadge(source: "hosted" | "own-key" | "subscription", usesCodexSubscription: boolean): string {
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
