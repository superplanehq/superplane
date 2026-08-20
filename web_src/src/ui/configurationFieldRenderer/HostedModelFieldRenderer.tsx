import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Text } from "@/components/Text/text";
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
  const { field, value, onChange, allValues, organizationId, readOnly = false } = props;
  const source = credentialsSource(allValues);
  const isHosted = source === "hosted";
  const provider = field.typeOptions?.hostedModel?.provider ?? "";
  const { data, isLoading } = useHostedLLMModels(organizationId, provider, isHosted);
  const models = [...(data?.models ?? [])].sort((left, right) =>
    compareModelLabels(left.name || left.id || "", right.name || right.id || ""),
  );
  const hostedEnabled = data?.enabled === true;

  if (!isHosted) {
    return <StringFieldRenderer {...props} />;
  }

  if (!organizationId) {
    return <div className="text-sm text-red-500 dark:text-red-400">This field requires organization context.</div>;
  }

  if (isLoading) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading SuperPlane-hosted models...</Text>;
  }

  if (!hostedEnabled || models.length === 0) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        SuperPlane-hosted models are not configured for this provider. Ask an installation admin to add a key and
        allowlist.
      </Text>
    );
  }

  const current = typeof value === "string" ? value : "";
  const testId = field.name ? toTestId(`field-${field.name}-hosted-model`) : undefined;

  return (
    <Select value={current} onValueChange={(next) => onChange(next || undefined)} disabled={readOnly}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={field.placeholder || "Select a SuperPlane-hosted model"} />
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
};
