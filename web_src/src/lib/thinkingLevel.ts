export const THINKING_LEVEL_KEY = "thinkingLevel";

export const THINKING_LEVEL_LOW = "low";
export const THINKING_LEVEL_MEDIUM = "medium";
export const THINKING_LEVEL_HIGH = "high";
const OMITTED_DRAFT_START_THINKING = "auto";
const STORED_THINKING_DEFAULT = "default";

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
  if (trimmed === "" || trimmed === OMITTED_DRAFT_START_THINKING) {
    return undefined;
  }
  return trimmed;
}

/** Word shown after a model name. Auto and an empty value stay hidden. */
export function visibleThinkingLevelLabel(value: string | undefined): string | undefined {
  const key = value?.trim() ?? "";
  if (key === "" || key === OMITTED_DRAFT_START_THINKING) {
    return undefined;
  }
  if (key === STORED_THINKING_DEFAULT) {
    return "Default";
  }
  return THINKING_LEVELS.find((level) => level.value === key)?.label;
}

/** Model name plus a thinking word. No model name means no word. */
export function modelNameWithThinking(modelName: string, thinkingLevel: string | undefined): string {
  const word = visibleThinkingLevelLabel(thinkingLevel);
  if (!modelName || !word) {
    return modelName;
  }
  return `${modelName} ${word}`;
}
