export type ReplayPayloadValidation = { valid: true; value: Record<string, unknown> } | { valid: false; error: string };

/**
 * A replay payload is sent as a protobuf Struct, which must be a JSON object at the
 * top level — a syntactically valid JSON array or scalar is still not acceptable.
 */
export function validateReplayPayload(text: string): ReplayPayloadValidation {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { valid: false, error: "Invalid JSON" };
  }

  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { valid: false, error: "Payload must be a JSON object" };
  }

  return { valid: true, value: parsed as Record<string, unknown> };
}
