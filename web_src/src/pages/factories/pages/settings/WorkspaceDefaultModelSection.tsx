import { useEffect, useMemo, useState } from "react";

import type { FactoriesFactory } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { useUpdateFactory } from "@/hooks/useFactoryData";
import {
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_HOSTED,
  type SelectableLLMModel,
} from "@/lib/selectableLLMModels";
import { showErrorToast } from "@/lib/toast";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

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
  const models = useMemo(() => modelsQuery.data ?? [], [modelsQuery.data]);
  const noModels = !modelsQuery.isLoading && models.length === 0;
  const savedKey = useMemo(() => savedDefaultKey(factory, models), [factory, models]);
  const [value, setValue] = useState(savedKey);

  useEffect(() => {
    if (updateFactory.isPending) {
      return;
    }
    setValue(savedKey);
  }, [savedKey, updateFactory.isPending]);

  const save = async (next: string) => {
    const previous = value;
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
        <Select value={value} onValueChange={(next) => void save(next)} disabled={!canUpdate || noModels}>
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
  const match = models.find((model) => {
    if (model.model.id !== modelId) {
      return false;
    }
    if (provider === "") {
      return model.source.id === SELECTABLE_LLM_SOURCE_BYOK;
    }
    return model.source.id === SELECTABLE_LLM_SOURCE_HOSTED && model.provider.id === provider;
  });
  return match?.key ?? NO_DEFAULT_MODEL;
}
