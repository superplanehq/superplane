import React from "react";
import { useParams } from "react-router";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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

function SuperPlaneModelField({ field, value, onChange, organizationId, readOnly = false }: FieldRendererProps) {
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
  const selected = superPlanePickerValue(
    value,
    hostedSelectableLLMModelKey(usage.data?.defaultHostedProvider ?? "", usage.data?.defaultHostedModel ?? ""),
    options,
  );

  return (
    <ModelSelect
      value={selected}
      placeholder={field.placeholder || "Instance SuperPlane agent model"}
      testId={field.name ? toTestId(`field-${field.name}-hosted-model`) : undefined}
      readOnly={readOnly}
      options={options}
      onChange={onChange}
    />
  );
}

function superPlanePickerValue(value: unknown, defaultKey: string, models: Array<{ value: string }>): string {
  const current = normalizeSuperPlaneModelValue(typeof value === "string" ? value : "");
  if (current !== "" && models.some((model) => model.value === current)) {
    return current;
  }
  if (models.some((model) => model.value === defaultKey)) {
    return defaultKey;
  }
  return "";
}

function ProviderBYOKModelField({ field, value, onChange, organizationId, readOnly = false }: FieldRendererProps) {
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
    <ModelSelect
      value={current}
      placeholder={field.placeholder || "Select a model"}
      testId={field.name ? toTestId(`field-${field.name}-hosted-model`) : undefined}
      readOnly={readOnly}
      options={options}
      onChange={onChange}
    />
  );
}

function ModelSelect({
  value,
  placeholder,
  testId,
  readOnly,
  options,
  onChange,
}: {
  value: string;
  placeholder: string;
  testId?: string;
  readOnly?: boolean;
  options: Array<{ value: string; label: string }>;
  onChange: (next: unknown) => void;
}) {
  return (
    <Select value={value || undefined} onValueChange={(next) => onChange(next || undefined)} disabled={readOnly}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent position="popper" className="max-h-60">
        {options.map((model) => (
          <SelectItem key={model.value} value={model.value}>
            {model.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
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
