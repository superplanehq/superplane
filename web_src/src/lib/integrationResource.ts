/**
 * An integration resource reference as returned by the API's
 * `integrations/.../resources` endpoint. Fields such as a node's `model` or
 * `repository` can be stored either as a plain string (the resource id/name)
 * or, once resolved from an integration during Setup, as this object shape.
 */
export interface IntegrationResourceRef {
  id?: string;
  name?: string;
  type?: string;
}

/**
 * Returns true when the value looks like an {@link IntegrationResourceRef}
 * (a plain object carrying an `id`, `name`, or `type`) rather than a scalar.
 */
export function isIntegrationResourceRef(value: unknown): value is IntegrationResourceRef {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return false;
  }

  const record = value as Record<string, unknown>;
  return "id" in record || "name" in record || "type" in record;
}

/**
 * Normalizes a value that may be a plain string or an
 * {@link IntegrationResourceRef} into a human-readable display string.
 *
 * Node metadata/config fields that back an integration-resource picker can hold
 * either shape. Rendering the raw `{ id, name, type }` object as a React child
 * crashes the whole canvas ("Objects are not valid as a React child"), so
 * callers must funnel such values through here before displaying them.
 *
 * @returns the display string, or `undefined` when nothing meaningful is present.
 */
export function resourceRefLabel(value: unknown): string | undefined {
  if (value === null || value === undefined) {
    return undefined;
  }

  if (typeof value === "string") {
    return value.length > 0 ? value : undefined;
  }

  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  if (isIntegrationResourceRef(value)) {
    return resourceRefLabel(value.name) ?? resourceRefLabel(value.id);
  }

  return undefined;
}
