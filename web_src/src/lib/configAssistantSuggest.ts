import type { Client } from "@/api-client/client/types.gen";
import { client } from "@/api-client/client.gen";
import { agentsPrepareConfigAssistantSuggest } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type ConfigAssistantSuggestPayload = {
  canvasId: string;
  nodeId: string;
  instruction: string;
  /** JSON string passed through to the agent (field metadata, current value, etc.). */
  fieldContextJson: string;
};

export type ConfigAssistantSuggestResult = {
  value: string;
  explanation?: string;
};

/**
 * Calls SuperPlane prepare-suggest (session auth), then POSTs to the agent HTTP service
 * with the minted Bearer token. Same contract as the former single-hop Go proxy.
 */
export async function runConfigAssistantSuggest(
  organizationId: string,
  payload: ConfigAssistantSuggestPayload,
  apiClient: Client = client,
): Promise<ConfigAssistantSuggestResult> {
  const prepareResponse = await agentsPrepareConfigAssistantSuggest(
    withOrganizationHeader({
      organizationId,
      client: apiClient,
      body: {
        canvasId: payload.canvasId,
      },
    }),
  );

  const token = prepareResponse.data?.token?.trim();
  const suggestUrl = prepareResponse.data?.suggestUrl?.trim();
  if (!token || !suggestUrl) {
    throw new Error("Config assistant prepare response missing token or suggest URL");
  }

  const res = await fetch(suggestUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      canvas_id: payload.canvasId,
      node_id: payload.nodeId,
      instruction: payload.instruction,
      field_context_json: payload.fieldContextJson,
    }),
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Config assistant request failed (${res.status})`);
  }

  const data = (await res.json()) as { value?: string; explanation?: string };
  const value = typeof data.value === "string" ? data.value.trim() : "";
  if (!value) {
    throw new Error("Config assistant returned an empty value");
  }

  const explanation = typeof data.explanation === "string" ? data.explanation.trim() : "";
  return {
    value,
    ...(explanation ? { explanation } : {}),
  };
}
