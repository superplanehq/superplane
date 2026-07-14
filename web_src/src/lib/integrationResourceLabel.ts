/**
 * Coerces an integration-resource configuration value to a display string.
 *
 * Configuration fields typed as `integration-resource` are usually stored as
 * id/name strings, but agent staging and some write paths can leave the full
 * IntegrationResourceRef object `{ id, name, type }` in node configuration.
 * Rendering that object as a React child throws.
 */
export function integrationResourceDisplayLabel(value: unknown): string | undefined {
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed || undefined;
  }

  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return undefined;
  }

  const record = value as { name?: unknown; id?: unknown };
  if (typeof record.name === "string" && record.name.trim()) {
    return record.name.trim();
  }
  if (typeof record.id === "string" && record.id.trim()) {
    return record.id.trim();
  }

  return undefined;
}
