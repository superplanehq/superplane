import React from "react";
import { ChevronDown } from "lucide-react";
import { useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/Text/text";
import { useCanvas } from "@/hooks/useCanvasData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";
import {
  byokRunnerModelOptions,
  hostedSelectableLLMModelKey,
  normalizeSuperPlaneModelValue,
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_HOSTED,
  selectableLLMModelsForProvider,
  type SelectableLLMSourceID,
} from "@/lib/selectableLLMModels";
import { toTestId } from "@/lib/testID";
import { THINKING_LEVEL_KEY, THINKING_LEVELS, normalizeThinkingLevel } from "@/lib/thinkingLevel";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { DropdownMenuValueSub } from "@/ui/dropdownMenu/DropdownMenuValueSub";
import type { FieldRendererProps } from "./types";
import { StringFieldRenderer } from "./StringFieldRenderer";

export const HostedModelFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  const provider = props.field.typeOptions?.hostedModel?.provider ?? "";
  if (provider === HOSTED_MODEL_ALL_PROVIDERS) {
    return <SuperPlaneModelField {...props} />;
  }
  if (provider !== "") {
    return <ProviderBYOKModelField {...props} />;
  }
  return <StringFieldRenderer {...props} />;
};

function useCanvasFactoryScope(organizationId: string | undefined) {
  const { appId } = useParams<{ appId?: string }>();
  const canvasQuery = useCanvas(organizationId ?? "", appId ?? "", {
    enabled: Boolean(organizationId && appId),
    staleTime: Infinity,
  });
  return {
    factoryId: canvasQuery.data?.metadata?.factoryId,
    waitingForCanvas: Boolean(appId) && canvasQuery.isPending,
  };
}

function SuperPlaneModelField({
  field,
  value,
  onChange,
  onValuesChange,
  allValues,
  organizationId,
  readOnly = false,
}: FieldRendererProps) {
  const selection = useSelectablePickerModels(organizationId, [SELECTABLE_LLM_SOURCE_HOSTED]);
  const usage = useOrganizationWorkspaceUsage(organizationId ?? "");
  const status = modelFieldStatus(organizationId, selection.isLoading || usage.isLoading, selection.isError);
  if (status) {
    return status;
  }
  if (selection.models.length === 0) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        No SuperPlane-hosted models are allowlisted. Ask an installation admin to add a key and select models.
      </Text>
    );
  }

  const options = selection.models.map((model) => ({ value: model.key, label: model.label }));
  const current = normalizeSuperPlaneModelValue(typeof value === "string" ? value : "");
  const selected = superPlanePickerValue(
    current,
    hostedSelectableLLMModelKey(usage.data?.defaultHostedProvider ?? "", usage.data?.defaultHostedModel ?? ""),
    options,
  );

  return (
    <ModelThinkingSelect
      fieldName={field.name}
      model={selected}
      committedModel={current}
      thinkingLevel={normalizeThinkingLevel(allValues?.[THINKING_LEVEL_KEY])}
      placeholder={field.placeholder || "Instance SuperPlane agent model"}
      readOnly={readOnly}
      options={options}
      onCommit={(nextModel, nextThinking) =>
        commitModelAndThinking(field.name, nextModel, nextThinking, onChange, onValuesChange)
      }
    />
  );
}

function superPlanePickerValue(current: string, defaultKey: string, models: Array<{ value: string }>): string {
  if (current !== "" && models.some((model) => model.value === current)) {
    return current;
  }
  if (models.some((model) => model.value === defaultKey)) {
    return defaultKey;
  }
  return "";
}

function ProviderBYOKModelField({
  field,
  value,
  onChange,
  onValuesChange,
  allValues,
  organizationId,
  readOnly = false,
}: FieldRendererProps) {
  const selection = useSelectablePickerModels(organizationId, [SELECTABLE_LLM_SOURCE_BYOK]);
  const status = modelFieldStatus(organizationId, selection.isLoading, selection.isError);
  if (status) {
    return status;
  }

  const current = typeof value === "string" ? value : "";
  const options = byokRunnerModelOptions(
    selectableLLMModelsForProvider(selection.models, field.typeOptions?.hostedModel?.provider ?? ""),
    current,
  );
  if (options.length === 0) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        No models are selected for this provider. Select models on Organization LLM Models, or connect a provider on
        Integrations.
      </Text>
    );
  }

  return (
    <ModelThinkingSelect
      fieldName={field.name}
      model={current}
      committedModel={current}
      thinkingLevel={normalizeThinkingLevel(allValues?.[THINKING_LEVEL_KEY])}
      placeholder={field.placeholder || "Select a model"}
      readOnly={readOnly}
      options={options}
      onCommit={(nextModel, nextThinking) =>
        commitModelAndThinking(field.name, nextModel, nextThinking, onChange, onValuesChange)
      }
    />
  );
}

function commitModelAndThinking(
  fieldName: string | undefined,
  model: string,
  thinkingLevel: string,
  onChange: (next: unknown) => void,
  onValuesChange?: (patch: Record<string, unknown>) => void,
) {
  const modelValue = model || undefined;
  const thinkingValue = thinkingLevel || undefined;
  if (onValuesChange && fieldName) {
    onValuesChange({ [fieldName]: modelValue, [THINKING_LEVEL_KEY]: thinkingValue });
    return;
  }
  onChange(modelValue);
}

function ModelThinkingSelect({
  fieldName,
  model,
  committedModel,
  thinkingLevel,
  placeholder,
  readOnly,
  options,
  onCommit,
}: {
  fieldName?: string;
  model: string;
  committedModel: string;
  thinkingLevel: string;
  placeholder: string;
  readOnly?: boolean;
  options: Array<{ value: string; label: string }>;
  onCommit: (model: string, thinkingLevel: string) => void;
}) {
  const selectedLabel = options.find((option) => option.value === model)?.label || model || placeholder;
  const modelListTestId = fieldName ? toTestId(`field-${fieldName}-hosted-model-list`) : undefined;
  const thinkingTestId = fieldName ? toTestId(`field-${fieldName}-hosted-thinking`) : undefined;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          disabled={readOnly}
          data-testid={fieldName ? toTestId(`field-${fieldName}-hosted-model`) : undefined}
          className="h-8 w-full justify-between px-2.5 font-normal shadow-xs"
        >
          <span className="truncate">{selectedLabel}</span>
          <ChevronDown className="size-4 opacity-50" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-44">
        <DropdownMenuValueSub
          label="Model"
          testId={modelListTestId}
          value={model}
          options={options}
          onValueChange={(nextModel) => onCommit(nextModel, thinkingLevel)}
        />
        <DropdownMenuValueSub
          label="Thinking"
          testId={thinkingTestId}
          value={thinkingLevel}
          options={[...THINKING_LEVELS]}
          onValueChange={(nextThinking) => onCommit(committedModel, nextThinking)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function modelFieldStatus(organizationId: string | undefined, isLoading: boolean, isError: boolean) {
  if (!organizationId) {
    return <div className="text-sm text-red-500 dark:text-red-400">This field requires organization context.</div>;
  }
  if (isLoading) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading models...</Text>;
  }
  if (isError) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Unable to load models. Try again.</Text>;
  }
  return null;
}

function useSelectablePickerModels(organizationId: string | undefined, sources: SelectableLLMSourceID[]) {
  const { factoryId, waitingForCanvas } = useCanvasFactoryScope(organizationId);
  const query = useSelectableLLMModels(organizationId, {
    factoryId,
    sources,
    enabled: Boolean(organizationId) && !waitingForCanvas,
  });
  return {
    isLoading: waitingForCanvas || query.isLoading,
    isError: Boolean(query.isError),
    models: query.data ?? [],
  };
}
