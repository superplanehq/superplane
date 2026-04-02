import type { ConfigurationField } from "@/api-client";
import { isConfigAssistantSupportedField } from "@/lib/configAssistantFields";
import { suggestConfigurationFieldOrMock } from "@/lib/configAssistantSuggest";
import type { SuggestFieldValueFn } from "@/ui/configurationFieldRenderer/types";
import { useCallback, useMemo } from "react";

export type UseConfigAssistantSuggestOptions = {
  organizationId: string | undefined;
  canvasId: string | undefined;
  nodeId: string | undefined;
  autocompleteExampleObj: Record<string, unknown> | null;
  /** Feature gate, e.g. `isInlineConfigAssistantEnabled() && !readOnly` */
  featureEnabled: boolean;
};

export type UseConfigAssistantSuggestResult = {
  /** Whether this field may show the assistant (feature on + type allowlist + not sensitive). */
  isFieldAssistantEnabled: (field: ConfigurationField) => boolean;
  /**
   * Returns a handler for this field, or undefined when the field should not use the assistant.
   * Pass getCurrentValue so each suggest uses the latest draft value (e.g. () => nodeConfiguration[name]).
   */
  getSuggestFieldValue: (field: ConfigurationField, getCurrentValue: () => unknown) => SuggestFieldValueFn | undefined;
};

/**
 * Centralizes config-assistant request shape, API vs mock fallback, and error mapping for inline field help.
 */
export function useConfigAssistantSuggest(options: UseConfigAssistantSuggestOptions): UseConfigAssistantSuggestResult {
  const { organizationId, canvasId, nodeId, autocompleteExampleObj, featureEnabled } = options;

  const ctx = useMemo(
    () => ({
      organizationId,
      canvasId,
      nodeId,
      autocompleteExample: autocompleteExampleObj,
    }),
    [organizationId, canvasId, nodeId, autocompleteExampleObj],
  );

  const isFieldAssistantEnabled = useCallback(
    (field: ConfigurationField) => featureEnabled && isConfigAssistantSupportedField(field),
    [featureEnabled],
  );

  const getSuggestFieldValue = useCallback(
    (field: ConfigurationField, getCurrentValue: () => unknown): SuggestFieldValueFn | undefined => {
      if (!isFieldAssistantEnabled(field)) {
        return undefined;
      }
      return async (instruction: string) =>
        suggestConfigurationFieldOrMock(ctx, {
          field,
          currentValue: getCurrentValue(),
          instruction,
        });
    },
    [ctx, autocompleteExampleObj, isFieldAssistantEnabled],
  );

  return { isFieldAssistantEnabled, getSuggestFieldValue };
}
