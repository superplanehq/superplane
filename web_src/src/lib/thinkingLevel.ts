export const THINKING_LEVEL_KEY = "thinkingLevel";

const OMITTED_DRAFT_START_THINKING = "auto";
const STORED_THINKING_DEFAULT = "default";

export type ThinkingOption = { value: string; label: string };

/** Every level that an agent CLI can receive, in display order. A model supports a subset. */
const THINKING_LEVEL_LABELS: ReadonlyArray<ThinkingOption> = [
  { value: "none", label: "None" },
  { value: "minimal", label: "Minimal" },
  { value: "low", label: "Low" },
  { value: "medium", label: "Medium" },
  { value: "high", label: "High" },
  { value: "xhigh", label: "Extra high" },
  { value: "max", label: "Max" },
];

const DEFAULT_THINKING_OPTION: ThinkingOption = { value: "", label: "Default" };

export function normalizeThinkingLevel(value: unknown): string {
  if (typeof value !== "string") {
    return "";
  }
  const key = value.trim().toLowerCase();
  return THINKING_LEVEL_LABELS.some((level) => level.value === key) ? key : "";
}

/** Known levels from a model catalog, in display order. */
export function modelThinkingLevels(levels: ReadonlyArray<string> | undefined): string[] {
  const keys = new Set((levels ?? []).map(normalizeThinkingLevel));
  return THINKING_LEVEL_LABELS.filter((level) => keys.has(level.value)).map((level) => level.value);
}

/**
 * Options for a model's Thinking menu. A model without levels has no options,
 * so the caller hides the menu. `withDefault` adds the runner Default first.
 */
export function thinkingOptions(
  levels: ReadonlyArray<string> | undefined,
  { withDefault }: { withDefault: boolean },
): ThinkingOption[] {
  const supported = modelThinkingLevels(levels);
  if (supported.length === 0) {
    return [];
  }
  const options = THINKING_LEVEL_LABELS.filter((level) => supported.includes(level.value));
  return withDefault ? [DEFAULT_THINKING_OPTION, ...options] : options;
}

/** Keeps a thinking level only when the model supports it. Otherwise the value is empty. */
export function supportedThinkingLevel(value: string, levels: ReadonlyArray<string> | undefined): string {
  const level = normalizeThinkingLevel(value);
  return level !== "" && modelThinkingLevels(levels).includes(level) ? level : "";
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
  return THINKING_LEVEL_LABELS.find((level) => level.value === key)?.label;
}

/** Model name plus a thinking word. No model name means no word. */
export function modelNameWithThinking(modelName: string, thinkingLevel: string | undefined): string {
  const word = visibleThinkingLevelLabel(thinkingLevel);
  if (!modelName || !word) {
    return modelName;
  }
  return `${modelName} ${word}`;
}
