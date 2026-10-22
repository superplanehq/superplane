import { useEffect, useMemo, useRef, useState } from "react";

import type { FactoriesFactory } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useUpdateFactory } from "@/hooks/useFactoryData";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { SELECTABLE_LLM_SOURCE_HOSTED, type SelectableLLMModel } from "@/lib/selectableLLMModels";
import { showErrorToast } from "@/lib/toast";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";
import {
  modelMatchesAgentSource,
  modelSourceFromIntegrationName,
  workspaceAgentModelSource,
} from "./workspaceAgentModelSource";

const MODEL_FIELD_ID = "agent-default-model-select";
const NO_DEFAULT_MODEL = "__none__";

export function WorkspaceDefaultModelSection({
  organizationId,
  factoryId,
  factory,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  factory?: FactoriesFactory | null;
  canUpdate: boolean;
}) {
  const modelsQuery = useSelectableLLMModels(organizationId, {
    factoryId,
    enabled: Boolean(organizationId && factoryId),
  });
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const sourceState = useWorkspaceAgentModelSource(organizationId, factory);
  const models = useMemo(
    () => (modelsQuery.data ?? []).filter((model) => modelMatchesAgentSource(model, sourceState.source)),
    [modelsQuery.data, sourceState.source],
  );
  const noModels = !modelsQuery.isLoading && !sourceState.isLoading && models.length === 0;
  const savedKey = useMemo(() => savedDefaultKey(factory, models), [factory, models]);
  const [value, setValue] = useState(savedKey);
  const saveLock = useRef(false);

  useEffect(() => {
    if (updateFactory.isPending) {
      return;
    }
    setValue(savedKey);
  }, [savedKey, updateFactory.isPending]);

  const save = async (next: string) => {
    if (saveLock.current || updateFactory.isPending) {
      return;
    }
    const previous = value;
    saveLock.current = true;
    setValue(next);
    try {
      if (next === NO_DEFAULT_MODEL) {
        await updateFactory.mutateAsync({ clearDefaultAgentModel: true });
        return;
      }
      const selected = models.find((model) => model.key === next);
      if (!selected) {
        setValue(previous);
        return;
      }
      await updateFactory.mutateAsync({
        defaultAgentModel: selected.model.id,
        defaultAgentModelProvider: selected.source.id === SELECTABLE_LLM_SOURCE_HOSTED ? selected.provider.id : "",
      });
    } catch {
      setValue(previous);
      showErrorToast(AGENT_RESOURCES_COPY.defaultModelSaveFailed);
    } finally {
      saveLock.current = false;
    }
  };

  return (
    <FactorySettingsCard data-testid="agent-default-model">
      <div className="mb-3 flex flex-col gap-1">
        <Label htmlFor={MODEL_FIELD_ID} className="cursor-pointer text-[13px] font-medium tracking-[-0.01em]">
          {AGENT_RESOURCES_COPY.defaultModelTitle}
        </Label>
        <p className="text-[12px] text-muted-foreground">
          {noModels ? AGENT_RESOURCES_COPY.defaultModelEmpty : AGENT_RESOURCES_COPY.defaultModelHelper}
        </p>
      </div>
      <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission} className="w-full">
        <Select
          value={value}
          onValueChange={(next) => void save(next)}
          disabled={!canUpdate || noModels || updateFactory.isPending || sourceState.isLoading}
        >
          <SelectTrigger id={MODEL_FIELD_ID} className="w-full" data-testid={MODEL_FIELD_ID}>
            <SelectValue placeholder={AGENT_RESOURCES_COPY.defaultModelPlaceholder} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_DEFAULT_MODEL}>{AGENT_RESOURCES_COPY.defaultModelNone}</SelectItem>
            {models.map((model) => (
              <SelectItem key={model.key} value={model.key}>
                {modelOptionLabel(model.source.name, model.label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </PermissionTooltip>
    </FactorySettingsCard>
  );
}

function modelOptionLabel(sourceName: string, label: string): string {
  if (!sourceName) {
    return label;
  }
  return `${sourceName}: ${label}`;
}

function savedDefaultKey(factory: FactoriesFactory | null | undefined, models: SelectableLLMModel[]): string {
  const modelId = factory?.defaultAgentModel?.trim() ?? "";
  if (!modelId) {
    return NO_DEFAULT_MODEL;
  }
  const provider = factory?.defaultAgentModelProvider?.trim() ?? "";
  const match = models.find((model) => model.model.id === modelId && modelMatchesSavedProvider(model, provider));
  return match?.key ?? NO_DEFAULT_MODEL;
}

function modelMatchesSavedProvider(model: SelectableLLMModel, provider: string): boolean {
  if (provider === "") {
    return model.source.id !== SELECTABLE_LLM_SOURCE_HOSTED;
  }
  return model.source.id === SELECTABLE_LLM_SOURCE_HOSTED && model.provider.id === provider;
}

const AGENT_MODEL_PROVIDERS = ["anthropic", "openai", "openrouter", "custom"] as const;

function needsAgentConnections(factory?: FactoriesFactory | null): boolean {
  const harness = factory?.onboarding?.agentHarness;
  return (
    Boolean(factory) &&
    harness !== "AGENT_HARNESS_SUPERPLANE" &&
    harness !== "AGENT_HARNESS_CODEX" &&
    harness !== "AGENT_HARNESS_CURSOR"
  );
}

function useAgentProviderQueries(organizationId: string, enabled: boolean) {
  return [
    useBYOKLLMModels(organizationId, "anthropic", enabled),
    useBYOKLLMModels(organizationId, "openai", enabled),
    useBYOKLLMModels(organizationId, "openrouter", enabled),
    useBYOKLLMModels(organizationId, "custom", enabled),
  ];
}

function useWorkspaceAgentModelSource(organizationId: string, factory?: FactoriesFactory | null) {
  const savedIntegrationId = factory?.onboarding?.agentIntegrationId?.trim() ?? "";
  const needsConnections = needsAgentConnections(factory);
  const needsSavedIntegration = needsConnections && savedIntegrationId !== "";
  const integrations = useConnectedIntegrations(organizationId, { enabled: needsSavedIntegration });
  const queries = useAgentProviderQueries(organizationId, needsConnections);
  const connections = AGENT_MODEL_PROVIDERS.map((provider, index) => ({
    provider,
    connected: Boolean(queries[index]?.data?.connected),
  }));
  const savedIntegration = integrations.data?.find((integration) => integration.metadata?.id === savedIntegrationId);
  const isLoading = !factory || queries.some((query) => query.isLoading) || integrations.isLoading;
  const source = isLoading
    ? undefined
    : (workspaceAgentModelSource(
        factory?.onboarding?.agentHarness,
        modelSourceFromIntegrationName(savedIntegration?.metadata?.integrationName),
        connections,
      ) ?? undefined);
  return { source, isLoading };
}
