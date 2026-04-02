import type { ConfigurationField } from "@/api-client";
import { configAssistantSuggestConfigurationField } from "@/api-client";
import { buildConfigAssistantFieldContext } from "@/lib/configAssistantFields";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type ConfigAssistantSuggestResult = {
  value: string;
  explanation?: string;
};

export type ConfigAssistantSuggestCallParams = {
  organizationId: string;
  canvasId: string;
  nodeId: string;
  instruction: string;
  field: ConfigurationField;
  currentValue: unknown;
  autocompleteExample: Record<string, unknown> | null;
};

const MOCK_DELAY_MS = 450;

const MOCK_FALLBACK_EXPLANATION = "Mock: enable the assistant and open a workflow node on a canvas to call the API.";

/**
 * Calls the public config-assistant API. Trims instruction; throws on empty model response or API errors.
 */
export async function suggestConfigurationField(
  params: ConfigAssistantSuggestCallParams,
): Promise<ConfigAssistantSuggestResult> {
  const instruction = params.instruction.trim();
  if (!instruction) {
    throw new Error("Instruction is required.");
  }

  try {
    const { data } = await configAssistantSuggestConfigurationField(
      withOrganizationHeader({
        organizationId: params.organizationId,
        body: {
          canvasId: params.canvasId,
          nodeId: params.nodeId,
          instruction,
          fieldContextJson: JSON.stringify({
            field: buildConfigAssistantFieldContext(params.field),
            currentValue: params.currentValue,
            autocompleteExample: params.autocompleteExample,
          }),
        },
      }),
    );
    const value = data?.value?.trim();
    if (!value) {
      throw new Error("The assistant returned an empty value.");
    }
    return {
      value,
      explanation: data?.explanation?.trim() || undefined,
    };
  } catch (err) {
    throw new Error(getApiErrorMessage(err, "Suggestion failed"));
  }
}

export type ConfigAssistantCallContext = {
  organizationId: string | undefined;
  canvasId: string | undefined;
  nodeId: string | undefined;
  autocompleteExample: Record<string, unknown> | null;
};

/**
 * If canvas/node/org and non-empty instruction are present, calls the API; otherwise dev mock (delay + echo).
 */
export async function suggestConfigurationFieldOrMock(
  ctx: ConfigAssistantCallContext,
  args: {
    field: ConfigurationField;
    currentValue: unknown;
    instruction: string;
  },
): Promise<ConfigAssistantSuggestResult> {
  const { organizationId, canvasId, nodeId, autocompleteExample } = ctx;
  const trimmedInstruction = args.instruction.trim();

  if (organizationId && canvasId && nodeId && trimmedInstruction.length > 0) {
    return suggestConfigurationField({
      organizationId,
      canvasId,
      nodeId,
      instruction: trimmedInstruction,
      field: args.field,
      currentValue: args.currentValue,
      autocompleteExample: autocompleteExample ?? null,
    });
  }

  await new Promise((resolve) => window.setTimeout(resolve, MOCK_DELAY_MS));
  return {
    value: trimmedInstruction.length > 0 ? trimmedInstruction : "true",
    explanation: MOCK_FALLBACK_EXPLANATION,
  };
}
