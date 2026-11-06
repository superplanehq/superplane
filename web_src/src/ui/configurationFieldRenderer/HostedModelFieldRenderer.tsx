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
  SELECTABLE_LLM_SOURCE_SUBSCRIPTION,
  selectableLLMModelLabelFromKey,
  selectableLLMModelsForProvider,
  type SelectableLLMModel,
  type SelectableLLMSourceID,
} from "@/lib/selectableLLMModels";
import { toTestId } from "@/lib/testID";
import {
  THINKING_LEVEL_KEY,
  normalizeThinkingLevel,
  supportedThinkingLevel,
  thinkingOptions,
  visibleThinkingLevelLabel,
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
  if (selection.models.length === 0 && !value) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        No SuperPlane models are available. Select models in Workspace Models or ask an administrator to configure a
        provider.
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
  if (current !== "") {
    return current;
  }
  if (models.some((model) => model.value === defaultKey)) {
    return defaultKey;
  }
  return "";
}

function SubscriptionModelField(props: FieldRendererProps) {
  const selection = useSelectablePickerModels(
    props.organizationId,
    [SELECTABLE_LLM_SOURCE_SUBSCRIPTION],
    props.factoryId,
  );
  const status = modelFieldStatus(props.organizationId, selection.isLoading, selection.isError);
  if (status) return status;
  const current = typeof props.value === "string" ? props.value : "";
  if (selection.models.length === 0 && current === "") {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        No ChatGPT models are available. Connect ChatGPT and select models in Workspace Models.
      </Text>
    );
  }
  return (
    <ModelThinkingSelect
      fieldName={props.field.name}
      model={current}
      committedModel={current}
      thinkingLevel={normalizeThinkingLevel(props.allValues?.[THINKING_LEVEL_KEY])}
      placeholder="Default"
      readOnly={props.readOnly}
      options={selection.models.map((model) => ({
        value: model.model.id,
        label: model.model.name,
        thinkingLevels: model.thinkingLevels,
        isDefault: model.isDefault,
      }))}
      modelOnly={props.modelOnly ?? false}
      unavailableSeverity="warning"
      unavailableMessage="Your Codex account cannot run this model. Choose an available model here or for a specific task."
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

  const { current, shown, options } = byokFieldSelection({ field, value, allValues }, selection.models);
  if (options.length === 0 && current === "") {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        No models are available for this provider. Select models in Workspace Models or connect a provider in
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

function byokFieldSelection(
  { field, value, allValues }: Pick<FieldRendererProps, "field" | "value" | "allValues">,
  models: SelectableLLMModel[],
) {
  const provider = byokModelProvider(field.typeOptions?.hostedModel?.provider ?? "", allValues?.llmProvider);
  const providerModels = selectableLLMModelsForProvider(models, provider);
  const current = typeof value === "string" ? value : "";
  const resolved = defaultByokRunnerModel(
    current,
    provider,
    providerModels.map((model) => model.model.id),
  );
  const options = byokRunnerModelOptions(providerModels).map((option) => ({
    value: option.value,
    label: shortClaudeModelLabel(option.value) ?? option.label,
    thinkingLevels: option.thinkingLevels,
  }));
  return { current, shown: resolved ?? current, options };
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

function modelThinkingPickerState(options: ModelOption[], model: string, thinkingLevel: string, modelOnly: boolean) {
  // An unset model runs as the catalog default, so show that model's levels.
  const effectiveModel = model !== "" ? model : (options.find((option) => option.isDefault)?.value ?? "");
  const levels = thinkingLevelsForModel(options, effectiveModel);
  const modelThinking = thinkingOptions(levels, { withDefault: true });
  const unavailableThinking = thinkingLevel !== "" && supportedThinkingLevel(thinkingLevel, levels) === "";
  if (unavailableThinking && modelThinking.length === 0) {
    modelThinking.push({ value: "", label: "Default" });
  }
  const showThinking = !modelOnly && (modelThinking.length > 0 || unavailableThinking);
  const thinkingLabel = unavailableThinking
    ? `${visibleThinkingLevelLabel(thinkingLevel)} (unavailable)`
    : (modelThinking.find((level) => level.value === thinkingLevel)?.label ?? "Default");
  return { modelThinking, showThinking, unavailableThinking, thinkingLabel };
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
  unavailableSeverity = "error",
  unavailableMessage = "Select an available model to replace the saved model.",
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
  unavailableSeverity?: "warning" | "error";
  unavailableMessage?: string;
  triggerClassName?: string;
  onCommit: (model: string, thinkingLevel: string) => void;
}) {
  const { unavailable, label: selectedLabel } = modelSelectionLabel(model, options, placeholder);
  const { modelThinking, showThinking, unavailableThinking, thinkingLabel } = modelThinkingPickerState(
    options,
    model,
    thinkingLevel,
    modelOnly,
  );
  const modelListTestId = modelFieldTestId(fieldName, "model-list");
  const thinkingTestId = modelFieldTestId(fieldName, "thinking");
  // A new model keeps the thinking level only when it supports that level.
  const commitModel = (nextModel: string) =>
    onCommit(nextModel, supportedThinkingLevel(thinkingLevel, thinkingLevelsForModel(options, nextModel)));

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="outline"
            disabled={readOnly || (options.length === 0 && !showThinking)}
            aria-invalid={(unavailable && unavailableSeverity === "error") || undefined}
            data-testid={modelFieldTestId(fieldName, "model")}
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
        <DropdownMenuContent
          align="start"
          className="min-w-44"
          data-testid={showThinking ? undefined : modelListTestId}
        >
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
                  value={thinkingLevel}
                  options={modelThinking}
                  disabled={unavailable}
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
      {unavailableThinking ? (
        <div>
          <Text role="status" className="text-sm text-amber-700 dark:text-amber-400">
            The saved thinking level is unavailable for this model. Select an available level or use Default.
          </Text>
          {!readOnly ? (
            <Button type="button" variant="link" onClick={() => onCommit(committedModel, "")}>
              Use default
            </Button>
          ) : null}
        </div>
      ) : null}
      {unavailable ? (
        <Text
          role={unavailableSeverity === "warning" ? "status" : "alert"}
          className={cn(
            "text-sm",
            unavailableSeverity === "warning" ? "text-amber-700 dark:text-amber-400" : "text-content-error",
          )}
        >
          {unavailableMessage}
        </Text>
      ) : null}
    </>
  );
}

function modelSelectionLabel(model: string, options: Array<{ value: string; label: string }>, placeholder: string) {
  const selected = options.find((option) => option.value === model);
  if (selected) return { unavailable: false, label: selected.label };
  if (model === "") return { unavailable: false, label: placeholder };
  const savedLabel = shortClaudeModelLabel(model) ?? selectableLLMModelLabelFromKey(model);
  return { unavailable: true, label: `${savedLabel} (unavailable)` };
}

function modelFieldTestId(fieldName: string | undefined, control: string) {
  return fieldName ? toTestId(`field-${fieldName}-hosted-${control}`) : undefined;
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
