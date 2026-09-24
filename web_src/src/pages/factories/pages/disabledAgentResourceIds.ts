export function disabledAgentResourceIds(configuration: Record<string, unknown> | undefined): string[] {
  const value = configuration?.disabledAgentResourceIds;
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((id): id is string => typeof id === "string" && id.trim() !== "");
}
