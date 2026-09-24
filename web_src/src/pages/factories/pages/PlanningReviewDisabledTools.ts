export function disabledAgentResourceTools(
  configuration: Record<string, unknown> | undefined,
): Record<string, string[]> {
  const value = configuration?.disabledAgentResourceTools;
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  const out: Record<string, string[]> = {};
  for (const [id, names] of Object.entries(value as Record<string, unknown>)) {
    if (!id.trim()) {
      continue;
    }
    if (!Array.isArray(names)) {
      continue;
    }
    out[id] = names
      .filter((name): name is string => typeof name === "string" && name.trim() !== "")
      .map((name) => name.trim());
  }
  return out;
}

export function nextDisabledAgentResourceTools(
  current: Record<string, string[]>,
  resourceId: string,
  toolName: string,
  enabled: boolean,
): Record<string, string[]> {
  const existing = current[resourceId] ?? [];
  const nextNames = enabled
    ? existing.filter((name) => name !== toolName)
    : existing.includes(toolName)
      ? existing
      : [...existing, toolName];
  const next = { ...current };
  if (nextNames.length === 0) {
    delete next[resourceId];
    return next;
  }
  next[resourceId] = nextNames;
  return next;
}
