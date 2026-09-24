export const THINKING_LEVEL_KEY = "thinkingLevel";

export const THINKING_LEVEL_LOW = "low";
export const THINKING_LEVEL_MEDIUM = "medium";
export const THINKING_LEVEL_HIGH = "high";
export const DRAFT_START_THINKING_AUTO = "auto";
export const DRAFT_START_THINKING_DEFAULT = "default";

export const THINKING_LEVELS = [
  { value: "", label: "Default" },
  { value: THINKING_LEVEL_LOW, label: "Low" },
  { value: THINKING_LEVEL_MEDIUM, label: "Medium" },
  { value: THINKING_LEVEL_HIGH, label: "High" },
] as const;

export function normalizeThinkingLevel(value: unknown): string {
  if (value === THINKING_LEVEL_LOW || value === THINKING_LEVEL_MEDIUM || value === THINKING_LEVEL_HIGH) {
    return value;
  }
  return "";
}

export function draftStartThinkingPayload(selected: string): string | undefined {
  const trimmed = selected.trim();
  if (trimmed === "" || trimmed === DRAFT_START_THINKING_AUTO) {
    return undefined;
  }
  return trimmed;
}
