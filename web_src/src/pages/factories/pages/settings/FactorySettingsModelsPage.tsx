import { PermissionTooltip } from "@/components/PermissionGate";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { useFactoryLLMModels, useUpdateFactoryLLMModels } from "@/hooks/useLLMModelAllowlists";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Switch } from "@/ui/switch";
import { Check, CircleDollarSign, Cpu, KeyRound } from "lucide-react";
import { useState } from "react";
import { useLocation } from "react-router";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { useFactoryAgent } from "./useFactoryAgent";
import {
  factoryAgentProviders,
  providerFor,
  resolveAgentModels,
  type AgentProvider,
  type CredentialSource,
  type ProviderOption,
  useFactoryAgentSelection,
} from "./useFactoryAgentSelection";

export function FactorySettingsModelsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const location = useLocation();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const onboarding = factory.onboarding;
  const [selections, setSelections] = useState<IntegrationSelections>({});
  const { data: integrations = [] } = useConnectedIntegrations(organizationId, { enabled: Boolean(organizationId) });
  const selection = useFactoryAgentSelection(onboarding, integrations);
  const {
    source,
    setSource,
    provider,
    setProvider,
    integrationId,
    setIntegrationId,
    selectedProvider,
    readyIntegrations,
    dirty,
  } = selection;
  const connect = useIntegrationConnectDialog({
    organizationId,
    returnTo: location.pathname,
    integrationNames: factoryAgentProviders.map((option) => option.integrationName),
    selections,
    onSelectionsChange: setSelections,
  });
  const { saveAgent, isPending: agentSavePending } = useSaveFactoryAgent({
    organizationId,
    factoryId,
    provider,
    source,
    integrationId,
  });
  const agentModels = useFactoryLLMModels(
    organizationId,
    factoryId,
    provider,
    source === "hosted" ? "hosted" : "byok",
    Boolean(organizationId && factoryId),
  );

  usePageTitle(["Models", "Settings", factory.name ?? "Workspace"]);

  const resolvedAgent = resolveAgentModels(selectedProvider, source, agentModels.data?.selected ?? []);

  return (
    <FactorySettingsPageFrame
      title="Models"
      subtitle="Choose the agent provider and credentials for generated workspace automations."
    >
      <FactorySettingsCard title="Workspace agent" data-testid="factory-settings-agent">
        <AgentSummary onboarding={onboarding} />
        <div className="mt-5 grid gap-3">
          <HostedOption active={source === "hosted"} provider={provider} onSelect={() => setSource("hosted")} />
          {factoryAgentProviders.map((option) => (
            <ProviderCard
              key={option.provider}
              active={source === "integration" && provider === option.provider}
              option={option}
              readyCount={countReadyIntegrations(integrations, option.integrationName)}
              onSelect={() => {
                setSource("integration");
                setProvider(option.provider);
                const next = integrations.find(
                  (integration) =>
                    integration.metadata?.integrationName === option.integrationName &&
                    integration.status?.state === "ready",
                );
                setIntegrationId(next?.metadata?.id ?? "");
              }}
              onConnect={() => connect.createNew(option.integrationName)}
            />
          ))}
        </div>

        <AgentCredentialSelection
          source={source}
          provider={provider}
          integrationId={integrationId}
          readyIntegrations={readyIntegrations}
          onProviderChange={setProvider}
          onIntegrationChange={setIntegrationId}
        />

        <AgentResolutionPreview source={source} provider={selectedProvider} resolution={resolvedAgent} />

        <div className="mt-4 flex items-center justify-between gap-4 border-t border-border pt-4">
          <Text className="max-w-xl text-[12px] text-muted-foreground">
            Saving updates generated Planning, Implementation, and Backlog runners. Current runs keep their existing
            version.
          </Text>
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You do not have permission to update this workspace."
          >
            <Button
              type="button"
              disabled={!dirty || !canUpdate || agentSavePending || (source === "integration" && !integrationId)}
              onClick={() => void saveAgent()}
              data-testid="factory-settings-agent-save"
            >
              {agentSavePending ? "Saving..." : "Save agent"}
            </Button>
          </PermissionTooltip>
        </div>
      </FactorySettingsCard>

      <FactorySettingsCard title="Available models">
        <Text className="text-[13px] text-muted-foreground">
          Choose which {hostedProviderLabel(provider)} models this workspace can use. The workspace agent uses the
          onboarding default.
        </Text>
        <FactoryProviderModelSection
          organizationId={organizationId}
          factoryId={factoryId}
          provider={provider}
          fundingSource={source === "hosted" ? "hosted" : "byok"}
          canUpdate={canUpdate}
        />
      </FactorySettingsCard>
      {connect.dialogs}
    </FactorySettingsPageFrame>
  );
}

function useSaveFactoryAgent({
  organizationId,
  factoryId,
  provider,
  source,
  integrationId,
}: {
  organizationId: string;
  factoryId: string;
  provider: AgentProvider;
  source: CredentialSource;
  integrationId: string;
}) {
  const updateAgent = useFactoryAgent(organizationId, factoryId);
  const saveAgent = async () => {
    if (source === "integration" && !integrationId) return;
    try {
      await updateAgent.mutateAsync({
        provider: providerToApi(provider),
        credentialSource:
          source === "hosted" ? "AGENT_CREDENTIAL_SOURCE_HOSTED" : "AGENT_CREDENTIAL_SOURCE_INTEGRATION",
        ...(source === "integration" ? { integrationId } : {}),
      });
      showSuccessToast("Workspace agent updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Unable to update the workspace agent."));
    }
  };
  return { saveAgent, isPending: updateAgent.isPending };
}

function AgentSummary({
  onboarding,
}: {
  onboarding: { agentProvider?: string; agentModel?: string; agentPlanningModel?: string } | undefined;
}) {
  const provider = providerFor(onboarding);
  const model = onboarding?.agentModel || "onboarding default";
  const planningModel = onboarding?.agentPlanningModel || model;
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-muted/20 p-4">
      <Cpu className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div>
        <p className="text-[13px] font-medium">
          {hostedProviderLabel(provider)} · {model} coding · {planningModel} planning
        </p>
        <p className="mt-1 text-[12px] text-muted-foreground">
          Planning uses the review model. Implementation and Backlog use the coding model.
        </p>
      </div>
    </div>
  );
}

function AgentResolutionPreview({
  source,
  provider,
  resolution,
}: {
  source: CredentialSource;
  provider: ProviderOption;
  resolution: { codingModel: string; planningModel: string };
}) {
  return (
    <div className="mt-4 rounded-lg border border-border bg-muted/20 p-4">
      <p className="text-[13px] font-medium">Will use {provider.runner}</p>
      <p className="mt-1 text-[12px] text-muted-foreground">
        {source === "hosted" ? "SuperPlane-hosted credit" : `${hostedProviderLabel(provider.provider)} key`} · coding{" "}
        {resolution.codingModel} · planning {resolution.planningModel}
      </p>
    </div>
  );
}

function AgentCredentialSelection({
  source,
  provider,
  integrationId,
  readyIntegrations,
  onProviderChange,
  onIntegrationChange,
}: {
  source: CredentialSource;
  provider: AgentProvider;
  integrationId: string;
  readyIntegrations: Array<{ metadata?: { id?: string; name?: string } }>;
  onProviderChange: (provider: AgentProvider) => void;
  onIntegrationChange: (integrationId: string) => void;
}) {
  if (source === "integration") {
    return (
      <div className="mt-4 rounded-lg border border-border bg-muted/20 p-4">
        <Label htmlFor="factory-agent-integration" className="text-[13px] font-medium">
          {hostedProviderLabel(provider)} key
        </Label>
        {readyIntegrations.length > 0 ? (
          <Select value={integrationId} onValueChange={onIntegrationChange}>
            <SelectTrigger id="factory-agent-integration" className="mt-2 h-9">
              <SelectValue placeholder={`Select a ${hostedProviderLabel(provider)} key`} />
            </SelectTrigger>
            <SelectContent>
              {readyIntegrations.map((integration) => (
                <SelectItem key={integration.metadata?.id} value={integration.metadata?.id ?? ""}>
                  {integration.metadata?.name || integration.metadata?.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Text className="mt-2 text-[13px] text-muted-foreground">
            Connect a {hostedProviderLabel(provider)} key to use this provider.
          </Text>
        )}
      </div>
    );
  }

  return (
    <div className="mt-4 rounded-lg border border-border bg-muted/20 p-4">
      <Label htmlFor="factory-agent-hosted-provider" className="text-[13px] font-medium">
        Hosted provider
      </Label>
      <Select value={provider} onValueChange={(value) => onProviderChange(value as AgentProvider)}>
        <SelectTrigger id="factory-agent-hosted-provider" className="mt-2 h-9">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {factoryAgentProviders.map((option) => (
            <SelectItem key={option.provider} value={option.provider}>
              {hostedProviderLabel(option.provider)} · {option.runner}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function HostedOption({
  active,
  provider,
  onSelect,
}: {
  active: boolean;
  provider: AgentProvider;
  onSelect: () => void;
}) {
  return (
    <button type="button" onClick={onSelect} className={optionClassName(active)} aria-pressed={active}>
      <CircleDollarSign className="mt-0.5 size-5 shrink-0" aria-hidden />
      <span className="min-w-0 flex-1 text-left">
        <span className="block text-[13px] font-medium">SuperPlane-hosted credit</span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">
          Use hosted {hostedProviderLabel(provider)} models for this workspace.
        </span>
      </span>
      {active ? <Check className="size-4 shrink-0" aria-hidden /> : null}
    </button>
  );
}

function ProviderCard({
  active,
  option,
  readyCount,
  onSelect,
  onConnect,
}: {
  active: boolean;
  option: ProviderOption;
  readyCount: number;
  onSelect: () => void;
  onConnect: () => void;
}) {
  return (
    <div className={optionClassName(active)}>
      <button
        type="button"
        onClick={onSelect}
        className="flex min-w-0 flex-1 items-start gap-3 text-left"
        aria-pressed={active}
      >
        <IntegrationIcon integrationName={option.integrationName} className="mt-0.5 size-5 shrink-0" size={20} />
        <span className="min-w-0 flex-1">
          <span className="block text-[13px] font-medium">{hostedProviderLabel(option.provider)}</span>
          <span className="mt-0.5 block text-[12px] text-muted-foreground">
            Use a {hostedProviderLabel(option.provider)} key with {option.runner}.
          </span>
        </span>
      </button>
      {readyCount > 0 ? (
        <span className="flex shrink-0 items-center gap-1.5 text-[12px] text-emerald-700 dark:text-emerald-300">
          {active ? <Check className="size-3.5" aria-hidden /> : <KeyRound className="size-3.5" aria-hidden />}
          {readyCount === 1 ? "Connected" : `${readyCount} connected`}
        </span>
      ) : (
        <Button type="button" variant="outline" size="sm" onClick={onConnect}>
          Connect
        </Button>
      )}
    </div>
  );
}

function optionClassName(active: boolean): string {
  return `flex items-start gap-3 rounded-lg border p-4 transition-colors ${
    active ? "border-foreground bg-accent/40" : "border-border bg-background hover:bg-accent/20"
  }`;
}

function providerToApi(
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

function countReadyIntegrations(
  integrations: Array<{ metadata?: { integrationName?: string }; status?: { state?: string } }>,
  name: string,
) {
  return integrations.filter(
    (integration) => integration.metadata?.integrationName === name && integration.status?.state === "ready",
  ).length;
}

function FactoryProviderModelSection({
  organizationId,
  factoryId,
  provider,
  fundingSource,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  provider: string;
  fundingSource: "hosted" | "byok";
  canUpdate: boolean;
}) {
  const factoryModels = useFactoryLLMModels(organizationId, factoryId, provider, fundingSource, true);
  const update = useUpdateFactoryLLMModels(organizationId, factoryId);
  const [query, setQuery] = useState("");
  const [inherit, setInherit] = useState<boolean | null>(null);
  const [draft, setDraft] = useState<string[] | null>(null);
  const parent = (factoryModels.data?.parent ?? []).map((model) => model.id ?? "").filter(Boolean);
  const inheritParent = inherit ?? factoryModels.data?.inheritParent !== false;
  const selected =
    draft ??
    (factoryModels.data?.selected ?? [])
      .map((model) => model.id ?? "")
      .filter((id) => id !== "" && parent.includes(id));

  const save = async () => {
    try {
      await update.mutateAsync({ provider, fundingSource, allowedModels: inheritParent ? [] : selected });
      setDraft(null);
      setInherit(null);
      showSuccessToast("Workspace models saved.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Unable to save workspace models."));
    }
  };

  const emptyMessage =
    fundingSource === "hosted"
      ? "No SuperPlane-hosted models are available."
      : "No organization models are selected for this provider.";

  return (
    <FactoryModelSectionBody
      canUpdate={canUpdate}
      emptyMessage={emptyMessage}
      inheritParent={inheritParent}
      isLoading={factoryModels.isLoading}
      isPending={update.isPending}
      parent={parent}
      query={query}
      selected={selected}
      onInheritChange={(checked) => {
        setInherit(checked);
        if (!checked) setDraft(selected.length > 0 ? selected : parent);
      }}
      onQueryChange={setQuery}
      onSave={() => void save()}
      onToggle={(model, checked) => setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model))}
    />
  );
}

function FactoryModelSectionBody({
  canUpdate,
  emptyMessage,
  inheritParent,
  isLoading,
  isPending,
  parent,
  query,
  selected,
  onInheritChange,
  onQueryChange,
  onSave,
  onToggle,
}: {
  canUpdate: boolean;
  emptyMessage: string;
  inheritParent: boolean;
  isLoading: boolean;
  isPending: boolean;
  parent: string[];
  query: string;
  selected: string[];
  onInheritChange: (checked: boolean) => void;
  onQueryChange: (query: string) => void;
  onSave: () => void;
  onToggle: (model: string, checked: boolean) => void;
}) {
  if (isLoading) return <Text className="mt-4 text-[13px] text-muted-foreground">Loading models...</Text>;
  if (parent.length === 0) return <Text className="mt-4 text-[13px] text-muted-foreground">{emptyMessage}</Text>;

  return (
    <div className="mt-4 space-y-3">
      <div className="flex items-center justify-between gap-3">
        <Label className="text-[13px]">Use organization list</Label>
        <Switch checked={inheritParent} disabled={!canUpdate || isPending} onCheckedChange={onInheritChange} />
      </div>
      {inheritParent ? (
        <Text className="text-[13px] text-muted-foreground">This workspace uses the organization model list.</Text>
      ) : (
        <ModelAllowlistEditor
          modelIds={parent}
          selected={selected}
          query={query}
          onQueryChange={onQueryChange}
          onToggle={onToggle}
          disabled={!canUpdate || isPending}
          searchLabel="Search available models"
        />
      )}
      <Button type="button" onClick={onSave} disabled={!canUpdate || isPending}>
        {isPending ? "Saving..." : "Save models"}
      </Button>
    </div>
  );
}
