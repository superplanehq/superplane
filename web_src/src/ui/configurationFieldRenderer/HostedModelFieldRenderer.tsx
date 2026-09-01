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

function SelectableModelField({
  field,
  value,
  onChange,
  organizationId,
  readOnly = false,
  fundingSource,
}: FieldRendererProps & { fundingSource: "hosted" | "byok" }) {
  const selection = useSelectableModels(organizationId, field.typeOptions?.hostedModel?.provider ?? "", fundingSource);

  if (!organizationId) {
    return <div className="text-sm text-red-500 dark:text-red-400">This field requires organization context.</div>;
  }
  if (selection.isLoading) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading models...</Text>;
  }
  if (selection.models.length === 0) {
    return <EmptyModelField fundingSource={fundingSource} />;
  }

  const current = typeof value === "string" ? value : "";
  const testId = field.name ? toTestId(`field-${field.name}-hosted-model`) : undefined;
  const placeholder = field.placeholder || defaultModelPlaceholder(fundingSource);

  return (
    <Select value={current} onValueChange={(next) => onChange(next || undefined)} disabled={readOnly}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent position="popper" className="max-h-60">
        {selection.models.map((model) => (
          <SelectItem key={model.id} value={model.id ?? ""}>
            {model.name || model.id}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function useSelectableModels(organizationId: string | undefined, provider: string, fundingSource: "hosted" | "byok") {
  const { factoryId, waitingForCanvas } = useCanvasFactoryScope(organizationId);
  const modelsReady = !waitingForCanvas;
  const hosted = useHostedLLMModels(organizationId, provider, fundingSource === "hosted" && modelsReady, factoryId);
  const byok = useBYOKLLMModels(organizationId, provider, fundingSource === "byok" && modelsReady, factoryId);
  const query = fundingSource === "hosted" ? hosted : byok;
  const models = (fundingSource === "hosted" ? (hosted.data?.models ?? []) : (byok.data?.selected ?? []))
    .slice()
    .sort((left, right) => compareModelLabels(left.name || left.id || "", right.name || right.id || ""));
  return { isLoading: waitingForCanvas || query.isLoading, models };
}

function EmptyModelField({ fundingSource }: { fundingSource: "hosted" | "byok" }) {
  const message =
    fundingSource === "hosted"
      ? "SuperPlane-hosted models are not configured for this provider. Ask an installation admin to add a key and allowlist."
      : "No models are selected for this provider. Select models on Organization Spending, or connect a provider key on Integrations.";
  return <Text className="text-sm text-gray-500 dark:text-gray-400">{message}</Text>;
}

function defaultModelPlaceholder(fundingSource: "hosted" | "byok") {
  if (fundingSource === "hosted") {
    return "Select a SuperPlane-hosted model";
  }
  return "Select a model";
}
