import { ArrowUpRight, Check, KeyRound } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";

import superplaneLogo from "@/assets/superplane.svg";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useUpdateBYOKLLMModels, type useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { type LLMModelsSwitchProvider, type LLMModelsSwitchTarget } from "./FactorySettingsLLMModelsSwitchPreview";
import {
  byokProviderProductName,
  ORGANIZATION_LLM_MODELS_COPY as COPY,
  providerKeyHeading,
  providerKeyIntegrationLink,
} from "./organizationLLMModelsCopy";
import { type WorkspaceModelSource } from "./workspaceModelSource";

type BYOKQuery = ReturnType<typeof useBYOKLLMModels>;

const BYOK_INTEGRATION_NAMES: Record<string, string> = {
  anthropic: "claude",
  openai: "openai",
  openrouter: "openrouter",
};

export function ModelSourceBody({
  source,
  switchedNotice,
  hostedModels,
  hostedLoading,
  hostedError,
  providers,
  currentProvider,
  organizationId,
  byokQueries,
  byokLoading,
  canUpdate,
  integrationsHref,
  onChoose,
}: {
  source: WorkspaceModelSource;
  switchedNotice: string | null;
  hostedModels: Array<{ key: string; label: string }>;
  hostedLoading: boolean;
  hostedError: boolean;
  providers: LLMModelsSwitchProvider[];
  currentProvider: LLMModelsSwitchProvider | null;
  organizationId: string;
  byokQueries: Record<string, BYOKQuery>;
  byokLoading: boolean;
  canUpdate: boolean;
  integrationsHref: string;
  onChoose: (target: LLMModelsSwitchTarget) => void;
}) {
  const notice = switchedNotice ? (
    <p className="text-xs text-muted-foreground" data-testid="llm-models-switched-notice">
      {switchedNotice}
    </p>
  ) : null;

  if (source === "hosted") {
    return (
      <div className="space-y-5">
        {notice}
        <HostedModelList models={hostedModels} isLoading={hostedLoading} isError={hostedError} />
        <ProviderChoices providers={providers} onChoose={onChoose} disabled={!canUpdate} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      {notice}
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
      <ChangeModelSource current={currentProvider} providers={providers} onChoose={onChoose} disabled={!canUpdate} />
    </div>
  );
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
              <ProviderMark provider={provider} />
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
  providers,
  onChoose,
  disabled,
}: {
  current: LLMModelsSwitchProvider | null;
  providers: LLMModelsSwitchProvider[];
  onChoose: (target: LLMModelsSwitchTarget) => void;
  disabled: boolean;
}) {
  const others = providers.filter((provider) => provider !== current);
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
              <ProviderMark provider={provider} />
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

function ProviderMark({ provider, className }: { provider: string; className?: string }) {
  if (provider === "custom") {
    return <KeyRound className={cn("size-4 shrink-0", className)} aria-hidden />;
  }
  return (
    <IntegrationIcon
      integrationName={BYOK_INTEGRATION_NAMES[provider] ?? provider}
      className={cn("size-4", className)}
      size={16}
    />
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
        <ProviderMark provider={provider} className="mt-0.5" />
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
  const allSelected = modelIds.length > 0 && modelIds.every((id) => selected.includes(id));

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
        onBulkToggle={() => setDraft(allSelected ? [] : [...modelIds])}
        disabled={!canUpdate || update.isPending}
        searchLabel={`Search ${byokProviderProductName(provider)} models`}
        showCount
        showBulkToggle
      />
      <PermissionTooltip allowed={canUpdate} message={COPY.noPermission}>
        <Button type="button" onClick={() => void save()} disabled={!canUpdate || update.isPending || draft === null}>
          {update.isPending ? COPY.saving : COPY.save}
        </Button>
      </PermissionTooltip>
    </div>
  );
}
