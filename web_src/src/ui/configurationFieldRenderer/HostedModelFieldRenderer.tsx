import React from "react";
import { useParams } from "react-router";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Text } from "@/components/Text/text";
import { useCanvas } from "@/hooks/useCanvasData";
import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { compareModelLabels } from "@/lib/hostedLLMModels";
import { toTestId } from "@/lib/testID";
import type { FieldRendererProps } from "./types";
import { StringFieldRenderer } from "./StringFieldRenderer";

function credentialsSource(allValues?: Record<string, unknown>): string {
  const credentials = allValues?.credentials;
  if (!credentials || typeof credentials !== "object" || Array.isArray(credentials)) {
    return "";
  }

  const source = (credentials as Record<string, unknown>).source;
  return typeof source === "string" ? source : "";
}

export const HostedModelFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  const source = credentialsSource(props.allValues);
  if (source === "hosted" || source === "secret" || source === "integration") {
    return <SelectableModelField {...props} fundingSource={source === "hosted" ? "hosted" : "byok"} />;
  }
  return <StringFieldRenderer {...props} />;
};

function useCanvasFactoryId(organizationId: string | undefined) {
  const { appId } = useParams<{ appId?: string }>();
  const canvasQuery = useCanvas(organizationId ?? "", appId ?? "", {
    enabled: Boolean(organizationId && appId),
    staleTime: Infinity,
  });
  return canvasQuery.data?.metadata?.factoryId;
}

function SelectableModelField({
  field,
  value,
  onChange,
  organizationId,
  readOnly = false,
  fundingSource,
}: FieldRendererProps & { fundingSource: "hosted" | "byok" }) {
  const provider = field.typeOptions?.hostedModel?.provider ?? "";
  const factoryId = useCanvasFactoryId(organizationId);
  const hosted = useHostedLLMModels(organizationId, provider, fundingSource === "hosted", factoryId);
  const byok = useBYOKLLMModels(organizationId, provider, fundingSource === "byok", factoryId);
  const isLoading = fundingSource === "hosted" ? hosted.isLoading : byok.isLoading;
  const models = (fundingSource === "hosted" ? (hosted.data?.models ?? []) : (byok.data?.selected ?? []))
    .slice()
    .sort((left, right) => compareModelLabels(left.name || left.id || "", right.name || right.id || ""));

  if (!organizationId) {
    return <div className="text-sm text-red-500 dark:text-red-400">This field requires organization context.</div>;
  }

  if (isLoading) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading models...</Text>;
  }

  if (models.length === 0) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        {fundingSource === "hosted"
          ? "SuperPlane-hosted models are not configured for this provider. Ask an installation admin to add a key and allowlist."
          : "No models are selected for this provider. Select models on LLM spend, or connect a provider key on Integrations."}
      </Text>
    );
  }

  const current = typeof value === "string" ? value : "";
  const testId = field.name ? toTestId(`field-${field.name}-hosted-model`) : undefined;
  const placeholder =
    fundingSource === "hosted"
      ? field.placeholder || "Select a SuperPlane-hosted model"
      : field.placeholder || "Select a model";

  return (
    <Select value={current} onValueChange={(next) => onChange(next || undefined)} disabled={readOnly}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent position="popper" className="max-h-60">
        {models.map((model) => (
          <SelectItem key={model.id} value={model.id ?? ""}>
            {model.name || model.id}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
