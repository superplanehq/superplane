export type LineColumnAutomations = Record<string, { canvasIds?: string[] }>;

export function columnAutomationCanvasIds(bindings: LineColumnAutomations | undefined, key: string): string[] {
  return (bindings?.[key]?.canvasIds ?? []).map((id) => id.trim()).filter((id): id is string => Boolean(id));
}

export function addColumnAutomation(
  current: LineColumnAutomations | undefined,
  key: string,
  canvasId: string,
): LineColumnAutomations {
  const id = canvasId.trim();
  const existing = columnAutomationCanvasIds(current, key);
  const next: LineColumnAutomations = { ...current };
  if (!id || existing.includes(id)) {
    if (existing.length > 0) {
      next[key] = { canvasIds: existing };
    }
    return next;
  }
  next[key] = { canvasIds: [...existing, id] };
  return next;
}

export function removeColumnAutomation(
  current: LineColumnAutomations | undefined,
  key: string,
  canvasId: string,
): LineColumnAutomations {
  const id = canvasId.trim();
  const remaining = columnAutomationCanvasIds(current, key).filter((entry) => entry !== id);
  const next: LineColumnAutomations = { ...current };
  if (remaining.length === 0) {
    delete next[key];
    return next;
  }
  next[key] = { canvasIds: remaining };
  return next;
}
