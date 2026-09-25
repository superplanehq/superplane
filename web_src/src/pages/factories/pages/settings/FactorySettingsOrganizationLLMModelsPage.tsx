import { ArrowUpRight, Check, KeyRound } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";

import superplaneLogo from "@/assets/superplane.svg";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import {
  BYOK_PROVIDERS,
  useBYOKLLMModels,
  useSwitchFactoryModelSource,
  useUpdateBYOKLLMModels,
} from "@/hooks/useLLMModelAllowlists";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { getApiErrorMessage } from "@/lib/errors";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { SELECTABLE_LLM_SOURCE_HOSTED } from "@/lib/selectableLLMModels";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import {
  type LLMModelsSwitchDialog,
  type LLMModelsSwitchProvider,
  type LLMModelsSwitchTarget,
  SwitchDialog,
} from "./FactorySettingsLLMModelsSwitchPreview";
import {
  byokProviderProductName,
  ORGANIZATION_LLM_MODELS_COPY as COPY,
  providerKeyHeading,
  providerKeyIntegrationLink,
} from "./organizationLLMModelsCopy";
import { workspaceModelSource } from "./workspaceModelSource";

type BYOKQuery = ReturnType<typeof useBYOKLLMModels>;

const BYOK_INTEGRATION_NAMES: Record<string, string> = {
  anthropic: "claude",
  openai: "openai",
  openrouter: "openrouter",
};

const SWITCH_PROVIDERS: LLMModelsSwitchProvider[] = ["anthropic", "openai", "openrouter"];

export function FactorySettingsOrganizationLLMModelsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("org", "update") && !permissionsLoading;
  const integrationsHref = useIntegrationsBasePath(organizationId);

  const anthropic = useBYOKLLMModels(organizationId, "anthropic", true);
  const openai = useBYOKLLMModels(organizationId, "openai", true);
  const openrouter = useBYOKLLMModels(organizationId, "openrouter", true);
  const byokQueries: Record<string, BYOKQuery> = { anthropic, openai, openrouter };
  const connected = BYOK_PROVIDERS.filter((provider) => {
    const query = byokQueries[provider];
    return query.data?.connected || (query.isError && !query.isLoading);
  });
  const byokLoading = BYOK_PROVIDERS.some((provider) => byokQueries[provider].isLoading);

  const hosted = useSelectableLLMModels(organizationId, { factoryId, sources: [SELECTABLE_LLM_SOURCE_HOSTED] });
  const source = workspaceModelSource(factory.onboarding?.agentHarness, connected.length > 0);
  const currentProvider = resolveCurrentProvider(factory.onboarding, connected, byokQueries);
  const [dialog, setDialog] = useState<LLMModelsSwitchDialog>(null);
  const [switchedNotice, setSwitchedNotice] = useState<string | null>(null);
  const switchSource = useSwitchFactoryModelSource(organizationId, factoryId);

  usePageTitle([COPY.pageTitle, "Settings", factory.name ?? "Workspace"]);

  const choose = (target: LLMModelsSwitchTarget) => {
    setDialog({ target, step: "warn" });
  };

  const saveSwitch = async (target: LLMModelsSwitchTarget, apiKey?: string) => {
    try {
      await switchSource.mutateAsync({ source: target, apiKey });
      setDialog(null);
      setSwitchedNotice(
        target === "hosted"
          ? "Automations in this workspace now use the SuperPlane agent."
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
              {source === "hosted" ? COPY.hostedBadge : COPY.ownKeyBadge}
            </Badge>
          }
        >
          {source === "hosted" ? (
            <div className="space-y-5">
              {switchedNotice ? (
                <p className="text-xs text-muted-foreground" data-testid="llm-models-switched-notice">
                  {switchedNotice}
                </p>
              ) : null}
              <HostedModelList
                models={hosted.data ?? []}
                isLoading={hosted.isLoading}
                isError={Boolean(hosted.isError)}
              />
              <ProviderChoices providers={SWITCH_PROVIDERS} onChoose={choose} disabled={!canUpdate} />
            </div>
          ) : (
            <div className="flex flex-col gap-5">
              {switchedNotice ? (
                <p className="text-xs text-muted-foreground" data-testid="llm-models-switched-notice">
                  {switchedNotice}
                </p>
              ) : null}
              {currentProvider ? (
                <OwnKeyModelLists
                  organizationId={organizationId}
                  connected={[currentProvider]}
                  byokQueries={byokQueries}
                  isLoading={byokLoading}
                  canUpdate={canUpdate}
                  integrationsHref={integrationsHref}
                />
              ) : (
                <NoProviderNotice integrationsHref={integrationsHref} />
              )}
              <ChangeModelSource current={currentProvider} onChoose={choose} disabled={!canUpdate} />
            </div>
          )}
        </FactorySettingsCard>
      </div>
      <SwitchDialog
        source={source === "hosted" ? "hosted" : (currentProvider ?? "anthropic")}
        dialog={dialog}
        onCancel={() => setDialog(null)}
        onContinueToKey={() => {
          if (dialog && dialog.target !== "hosted") {
            setDialog({ target: dialog.target, step: "key" });
          }
        }}
        onBackToWarning={() => {
          if (dialog) {
            setDialog({ target: dialog.target, step: "warn" });
          }
        }}
        onSaveKey={(apiKey) => {
          if (dialog && dialog.target !== "hosted") {
            void saveSwitch(dialog.target, apiKey);
          }
        }}
        onSwitchToHosted={() => {
          void saveSwitch("hosted");
        }}
      />
    </FactorySettingsPageFrame>
  );
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
  return value === "anthropic" || value === "openai" || value === "openrouter";
}

function ProviderChoices({
  providers,
  onChoose,
  disabled,
}: {
  providers: LLMModelsSwitchProvider[];
  onChoose: (target: LLMModelsSwitchTarget) => void;
  disabled: boolean;
}) {
  return (
    <div className="space-y-3 border-t border-border pt-4" data-testid="llm-models-provider-choices">
      <div>
        <h3 className="text-[13px] font-medium text-foreground">Use your own key</h3>
        <p className="mt-1 text-xs text-muted-foreground">Connect a provider key. The provider bills these runs.</p>
      </div>
      <ul className="space-y-2">
        {providers.map((provider) => (
          <li
            key={provider}
            className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
            data-testid={`llm-models-connect-${provider}`}
          >
            <span className="flex min-w-0 items-center gap-2 text-[13px]">
              <IntegrationIcon
                integrationName={BYOK_INTEGRATION_NAMES[provider] ?? provider}
                className="size-4"
                size={16}
              />
              {byokProviderProductName(provider)}
            </span>
            <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={() => onChoose(provider)}>
              Connect
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function ChangeModelSource({
  current,
  onChoose,
  disabled,
}: {
  current: LLMModelsSwitchProvider | null;
  onChoose: (target: LLMModelsSwitchTarget) => void;
  disabled: boolean;
}) {
  const others = SWITCH_PROVIDERS.filter((provider) => provider !== current);
  return (
    <div className="space-y-3 border-t border-border pt-4" data-testid="llm-models-change-source">
      <div>
        <h3 className="text-[13px] font-medium text-foreground">Change model source</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          Switching replaces the {current ? byokProviderProductName(current) : "current"} agent in every automation in
          this workspace.
        </p>
      </div>
      <ul className="space-y-2">
        <li className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2">
          <span className="flex min-w-0 items-center gap-2 text-[13px]">
            <img src={superplaneLogo} alt="" className="size-4 dark:brightness-0 dark:invert" />
            SuperPlane
          </span>
          <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={() => onChoose("hosted")}>
            Use SuperPlane
          </Button>
        </li>
        {others.map((provider) => (
          <li
            key={provider}
            className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
            data-testid={`llm-models-connect-${provider}`}
          >
            <span className="flex min-w-0 items-center gap-2 text-[13px]">
              <IntegrationIcon
                integrationName={BYOK_INTEGRATION_NAMES[provider] ?? provider}
                className="size-4"
                size={16}
              />
              {byokProviderProductName(provider)}
            </span>
            <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={() => onChoose(provider)}>
              Connect
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function StatusText({ children }: { children: string }) {
  return <p className="text-[13px] text-muted-foreground">{children}</p>;
}

function HostedModelList({
  models,
  isLoading,
  isError,
}: {
  models: Array<{ key: string; label: string }>;
  isLoading: boolean;
  isError: boolean;
}) {
  if (isLoading) return <StatusText>{COPY.loading}</StatusText>;
  if (isError) return <StatusText>{COPY.listError}</StatusText>;
  if (models.length === 0) return <StatusText>{COPY.hostedModelsEmpty}</StatusText>;

  return (
    <div className="space-y-3" data-testid="llm-models-hosted">
      <p className="text-xs text-muted-foreground">{COPY.hostedModelsHelper}</p>
      <ul className="max-h-56 space-y-2 overflow-auto">
        {models.map((model) => (
          <li key={model.key} className="flex items-center gap-2 text-[13px]">
            <Check className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            <span className="font-mono text-xs">{model.label}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function OwnKeyModelLists({
  organizationId,
  connected,
  byokQueries,
  isLoading,
  canUpdate,
  integrationsHref,
}: {
  organizationId: string;
  connected: string[];
  byokQueries: Record<string, BYOKQuery>;
  isLoading: boolean;
  canUpdate: boolean;
  integrationsHref: string;
}) {
  if (isLoading && connected.length === 0) return <StatusText>{COPY.loading}</StatusText>;
  if (connected.length === 0) return <NoProviderNotice integrationsHref={integrationsHref} />;

  return (
    <div className="flex flex-col gap-5">
      {connected.map((provider, index) => (
        <div
          key={provider}
          className={cn("space-y-3", index > 0 && "border-t border-border pt-4")}
          data-testid={`llm-models-provider-${provider}`}
        >
          <ProviderKeyHeader
            provider={provider}
            integrationHref={integrationDetailHref(integrationsHref, byokQueries[provider]?.data?.integrationId)}
          />
          <ProviderModelAllowlist
            organizationId={organizationId}
            provider={provider}
            query={byokQueries[provider]}
            canUpdate={canUpdate}
          />
        </div>
      ))}
    </div>
  );
}

function integrationDetailHref(integrationsHref: string, integrationId: string | undefined): string {
  return integrationId ? `${integrationsHref}/${integrationId}` : integrationsHref;
}

function ProviderKeyHeader({ provider, integrationHref }: { provider: string; integrationHref: string }) {
  return (
    <div
      className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-muted/40 px-3 py-2"
      data-testid={`llm-models-key-${provider}`}
    >
      <div className="flex min-w-0 items-start gap-2">
        <IntegrationIcon
          integrationName={BYOK_INTEGRATION_NAMES[provider] ?? provider}
          className="mt-0.5 size-4 shrink-0"
          size={16}
        />
        <div className="min-w-0">
          <p className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{providerKeyHeading(provider)}</p>
          <p className="text-[12px] text-muted-foreground">{COPY.ownKeyHelper}</p>
        </div>
      </div>
      <Button asChild variant="outline" size="sm">
        <Link to={integrationHref}>
          {providerKeyIntegrationLink(provider)}
          <ArrowUpRight className="size-3.5" aria-hidden />
        </Link>
      </Button>
    </div>
  );
}

function NoProviderNotice({ integrationsHref }: { integrationsHref: string }) {
  return (
    <div
      role="status"
      data-testid="llm-models-empty-banner"
      className="flex w-full flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-muted/40 px-3 py-2 text-[13px] text-muted-foreground"
    >
      <div className="flex min-w-0 items-start gap-2">
        <KeyRound className="mt-0.5 size-4 shrink-0" aria-hidden />
        <p>
          <span className="font-medium text-foreground">{COPY.noProviderTitle}. </span>
          {COPY.noProviderDescription}
        </p>
      </div>
      <Button asChild variant="outline" size="sm">
        <Link to={integrationsHref}>{COPY.noProviderAction}</Link>
      </Button>
    </div>
  );
}

function modelIdsOf(models: Array<{ id?: string }> | undefined): string[] {
  return (models ?? []).map((model) => model.id ?? "").filter(Boolean);
}

function ProviderModelAllowlist({
  organizationId,
  provider,
  query: providerQuery,
  canUpdate,
}: {
  organizationId: string;
  provider: string;
  query: BYOKQuery | undefined;
  canUpdate: boolean;
}) {
  const savedIds = modelIdsOf(providerQuery?.data?.selected);
  const candidates = modelIdsOf(providerQuery?.data?.candidates);
  const modelIds = candidates.length > 0 ? candidates : savedIds;

  if (providerQuery?.isLoading) return <StatusText>{COPY.loading}</StatusText>;
  if (providerQuery?.error) return <StatusText>{COPY.listError}</StatusText>;
  if (modelIds.length === 0) return <StatusText>{COPY.emptyCatalog}</StatusText>;

  return (
    <ProviderModelEditor
      organizationId={organizationId}
      provider={provider}
      modelIds={modelIds}
      savedIds={savedIds}
      canUpdate={canUpdate}
    />
  );
}

function ProviderModelEditor({
  organizationId,
  provider,
  modelIds,
  savedIds,
  canUpdate,
}: {
  organizationId: string;
  provider: string;
  modelIds: string[];
  savedIds: string[];
  canUpdate: boolean;
}) {
  const update = useUpdateBYOKLLMModels(organizationId);
  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState<string[] | null>(null);
  const selected = draft ?? savedIds;

  const save = async () => {
    try {
      await update.mutateAsync({ provider, allowedModels: selected });
      setDraft(null);
      showSuccessToast(COPY.saveSuccess);
    } catch (saveError) {
      showErrorToast(getApiErrorMessage(saveError, COPY.saveError));
    }
  };

  return (
    <div className="space-y-3">
      <ModelAllowlistEditor
        modelIds={modelIds}
        selected={selected}
        query={search}
        onQueryChange={setSearch}
        onToggle={(model, checked) => setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model))}
        disabled={!canUpdate || update.isPending}
        searchLabel={`Search ${byokProviderProductName(provider)} models`}
        showCount
      />
      <PermissionTooltip allowed={canUpdate} message={COPY.noPermission}>
        <Button type="button" onClick={() => void save()} disabled={!canUpdate || update.isPending || draft === null}>
          {update.isPending ? COPY.saving : COPY.save}
        </Button>
      </PermissionTooltip>
    </div>
  );
}
