import React from "react";
import { Check, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/Text/text";
import { preferredFactoryScope, useCanvasFactoryScope } from "@/hooks/useCanvasFactoryScope";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { HOSTED_MODEL_ALL_PROVIDERS, shortClaudeModelLabel } from "@/lib/hostedLLMModels";
import {
  byokRunnerModelOptions,
  defaultByokRunnerModel,
  hostedSelectableLLMModelKey,
  normalizeSuperPlaneModelValue,
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_HOSTED,
  selectableLLMModelsForProvider,
  type SelectableLLMSourceID,
} from "@/lib/selectableLLMModels";
import { toTestId } from "@/lib/testID";
import { useFactoryCodexConnection } from "@/pages/factories/pages/onboarding/useFactoryCodexConnection";
import {
  THINKING_LEVEL_KEY,
  normalizeThinkingLevel,
  supportedThinkingLevel,
  thinkingOptions,
} from "@/lib/thinkingLevel";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { DropdownMenuValueSub } from "@/ui/dropdownMenu/DropdownMenuValueSub";
import type { FieldRendererProps } from "./types";
import { StringFieldRenderer } from "./StringFieldRenderer";

/**
 * One model in a picker. `thinkingLevels` comes from the model catalog.
 * `isDefault` marks the model that the provider uses when the saved model is unset.
 */
type ModelOption = { value: string; label: string; thinkingLevels?: string[]; isDefault?: boolean };

export const HostedModelFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  const provider = props.field.typeOptions?.hostedModel?.provider ?? "";
  if (provider === HOSTED_MODEL_ALL_PROVIDERS) {
    return <SuperPlaneModelField {...props} />;
  }
  if (provider !== "") {
    const credentials = props.allValues?.credentials;
    if (
      provider === "openai" &&
      typeof credentials === "object" &&
      credentials !== null &&
      "source" in credentials &&
      credentials.source === "subscription"
    ) {
      return <SubscriptionModelField {...props} />;
    }
    return <ProviderBYOKModelField {...props} />;
  }
  return <StringFieldRenderer {...props} />;
};

function SuperPlaneModelField({
  field,
  value,
  onChange,
  onValuesChange,
  allValues,
  organizationId,
  factoryId,
  readOnly = false,
  triggerClassName,
  modelOnly = false,
}: FieldRendererProps) {
  const selection = useSelectablePickerModels(organizationId, [SELECTABLE_LLM_SOURCE_HOSTED], factoryId);
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

  const options = selection.models.map((model) => ({
    value: model.key,
    label: model.label,
    thinkingLevels: model.thinkingLevels,
  }));
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
      modelOnly={modelOnly}
      triggerClassName={triggerClassName}
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

function SubscriptionModelField(props: FieldRendererProps) {
  const canvasScope = useCanvasFactoryScope(props.organizationId);
  const { factoryId, waitingForCanvas } = preferredFactoryScope(props.factoryId, canvasScope);
  const { query } = useFactoryCodexConnection(props.organizationId ?? "", factoryId ?? "", !waitingForCanvas);
  const status = modelFieldStatus(props.organizationId, waitingForCanvas || query.isLoading, query.isError);
  if (status) return status;
  const connection = query.data;
  const options = connection?.status === "ready" ? subscriptionModelOptions(connection.availableModelOptions) : [];
  if (options.length === 0) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        No ChatGPT models are available. Connect ChatGPT in workspace setup to load models.
      </Text>
    );
  }
  const current = typeof props.value === "string" ? props.value : "";
  return (
    <ModelThinkingSelect
      fieldName={props.field.name}
      model={current}
      committedModel={current}
      thinkingLevel={normalizeThinkingLevel(props.allValues?.[THINKING_LEVEL_KEY])}
      placeholder="Default"
      readOnly={props.readOnly}
      options={options}
      modelOnly={props.modelOnly ?? false}
      triggerClassName={props.triggerClassName}
      onCommit={(model, thinking) =>
        commitModelAndThinking(props.field.name, model, thinking, props.onChange, props.onValuesChange)
      }
    />
  );
}

function ProviderBYOKModelField({
  field,
  value,
  onChange,
  onValuesChange,
  allValues,
  organizationId,
  factoryId,
  readOnly = false,
  triggerClassName,
  modelOnly = false,
}: FieldRendererProps) {
  const selection = useSelectablePickerModels(organizationId, [SELECTABLE_LLM_SOURCE_BYOK], factoryId);
  const status = modelFieldStatus(organizationId, selection.isLoading, selection.isError);
  if (status) {
    return status;
  }

  const provider = byokModelProvider(field.typeOptions?.hostedModel?.provider ?? "", allValues?.llmProvider);
  const providerModels = selectableLLMModelsForProvider(selection.models, provider);
  const current = typeof value === "string" ? value : "";
  const resolved = defaultByokRunnerModel(
    current,
    provider,
    providerModels.map((model) => model.model.id),
  );
  const shown = resolved ?? current;
  const options = byokRunnerModelOptions(providerModels, shown).map((option) => ({
    ...option,
    label: shortClaudeModelLabel(option.value) ?? option.label,
  }));
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
      model={shown}
      committedModel={current.trim() === "" ? current : shown}
      thinkingLevel={normalizeThinkingLevel(allValues?.[THINKING_LEVEL_KEY])}
      placeholder={field.placeholder || "Select a model"}
      readOnly={readOnly}
      options={options}
      modelOnly={modelOnly}
      triggerClassName={triggerClassName}
      onCommit={(nextModel, nextThinking) =>
        commitModelAndThinking(field.name, nextModel, nextThinking, onChange, onValuesChange)
      }
    />
  );
}

function subscriptionModelOptions(
  models: Array<{ id?: string; name?: string; thinkingLevels?: string[]; isDefault?: boolean }> | undefined,
) {
  return (models ?? [])
    .filter((model) => (model.id ?? "") !== "")
    .map((model) => ({
      value: model.id ?? "",
      label: model.name || model.id || "",
      thinkingLevels: model.thinkingLevels,
      isDefault: model.isDefault,
    }));
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

function thinkingLevelsForModel(options: ModelOption[], model: string) {
  return options.find((option) => option.value === model)?.thinkingLevels;
}

function modelThinkingPickerState(
  options: ModelOption[],
  model: string,
  thinkingLevel: string,
  placeholder: string,
  modelOnly: boolean,
) {
  const selectedLabel = options.find((option) => option.value === model)?.label || model || placeholder;
  // An unset model runs as the catalog default, so show that model's levels.
  const effectiveModel = model !== "" ? model : (options.find((option) => option.isDefault)?.value ?? "");
  const levels = thinkingLevelsForModel(options, effectiveModel);
  const modelThinking = thinkingOptions(levels, { withDefault: true });
  const showThinking = !modelOnly && modelThinking.length > 0;
  const selectedThinking = supportedThinkingLevel(thinkingLevel, levels);
  const thinkingLabel = modelThinking.find((level) => level.value === selectedThinking)?.label ?? "Default";
  return { selectedLabel, modelThinking, showThinking, selectedThinking, thinkingLabel };
}

function ModelThinkingSelect({
  fieldName,
  model,
  committedModel,
  thinkingLevel,
  placeholder,
  readOnly,
  options,
  modelOnly,
  triggerClassName,
  onCommit,
}: {
  fieldName?: string;
  model: string;
  committedModel: string;
  thinkingLevel: string;
  placeholder: string;
  readOnly?: boolean;
  options: ModelOption[];
  modelOnly: boolean;
  triggerClassName?: string;
  onCommit: (model: string, thinkingLevel: string) => void;
}) {
  const { selectedLabel, modelThinking, showThinking, selectedThinking, thinkingLabel } = modelThinkingPickerState(
    options,
    model,
    thinkingLevel,
    placeholder,
    modelOnly,
  );
  const modelListTestId = fieldName ? toTestId(`field-${fieldName}-hosted-model-list`) : undefined;
  const thinkingTestId = fieldName ? toTestId(`field-${fieldName}-hosted-thinking`) : undefined;
  // A new model keeps the thinking level only when it supports that level.
  const commitModel = (nextModel: string) =>
    onCommit(nextModel, supportedThinkingLevel(thinkingLevel, thinkingLevelsForModel(options, nextModel)));

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          disabled={readOnly}
          data-testid={fieldName ? toTestId(`field-${fieldName}-hosted-model`) : undefined}
          className={cn(
            "h-8 w-full justify-between !rounded-md border-gray-300 bg-white px-2.5 font-normal shadow-xs dark:border-gray-600/70 dark:bg-gray-800",
            triggerClassName,
          )}
        >
          <span className="min-w-0 truncate">{selectedLabel}</span>
          {showThinking ? (
            <span className="ml-auto shrink-0 text-[11px] text-muted-foreground">{thinkingLabel}</span>
          ) : null}
          <ChevronDown className="size-4 opacity-50" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-44" data-testid={showThinking ? undefined : modelListTestId}>
        {showThinking
          ? [
              <DropdownMenuValueSub
                key="model"
                label="Model"
                testId={modelListTestId}
                value={model}
                options={options}
                onValueChange={commitModel}
              />,
              <DropdownMenuValueSub
                key="thinking"
                label="Thinking"
                testId={thinkingTestId}
                value={selectedThinking}
                options={modelThinking}
                onValueChange={(nextThinking) => onCommit(committedModel, nextThinking)}
              />,
            ]
          : options.map((option) => (
              <DropdownMenuItem key={option.value} onSelect={() => commitModel(option.value)}>
                {option.label}
                {option.value === model ? <Check className="ml-auto size-4" aria-hidden /> : null}
              </DropdownMenuItem>
            ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** OpenCode agents store llmProvider "custom" while the component field still says openrouter. */
function byokModelProvider(fieldProvider: string, llmProvider: unknown): string {
  if (typeof llmProvider === "string" && llmProvider.trim() === "custom") {
    return "custom";
  }
  return fieldProvider;
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

function useSelectablePickerModels(
  organizationId: string | undefined,
  sources: SelectableLLMSourceID[],
  explicitFactoryId?: string,
) {
  const canvasScope = useCanvasFactoryScope(organizationId);
  const { factoryId, waitingForCanvas } = preferredFactoryScope(explicitFactoryId, canvasScope);
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
