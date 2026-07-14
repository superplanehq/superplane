/**
 * Coerces a configuration/metadata value coming from an `integration-resource`
 * field into a safe display string.
 *
 * Those fields are normally saved as a plain id/name string, but the full
 * `IntegrationResourceRef` shape (`{ id, name, type }`) can end up in node
 * configuration or metadata through other write paths (agent-set configuration,
 * spec import, stale drafts, etc.). Rendering that object directly as a React
 * child throws "Objects are not valid as a React child", so every place that
 * displays one of these fields must go through this helper instead of trusting
 * the value's declared type.
 */
export function integrationResourceDisplayLabel(value: unknown): string | undefined {
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed || undefined;
  }

  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return undefined;
  }

  const { name, id } = value as { name?: unknown; id?: unknown };
  if (typeof name === "string" && name.trim()) {
    return name.trim();
  }
  if (typeof id === "string" && id.trim()) {
    return id.trim();
  }

  return undefined;
}
