export const THINKING_LEVEL_KEY = "thinkingLevel";

const OMITTED_DRAFT_START_THINKING = "auto";
const STORED_THINKING_DEFAULT = "default";

export type ThinkingOption = { value: string; label: string };

/** Display labels and order for familiar levels. Model catalogs can supply more. */
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
  if (key === OMITTED_DRAFT_START_THINKING || key === STORED_THINKING_DEFAULT) {
    return "";
  }
  return /^[a-z][a-z0-9_-]*$/.test(key) ? key : "";
}

/** Catalog levels, with familiar levels first and additional efforts in catalog order. */
export function modelThinkingLevels(levels: ReadonlyArray<string> | undefined): string[] {
  const keys = new Set((levels ?? []).map(normalizeThinkingLevel));
  keys.delete("");
  const known = THINKING_LEVEL_LABELS.filter((level) => keys.has(level.value)).map((level) => level.value);
  return [...known, ...[...keys].filter((level) => !known.includes(level))];
}

function thinkingLevelLabel(level: string): string {
  const known = THINKING_LEVEL_LABELS.find((option) => option.value === level);
  if (known) return known.label;
  const words = level.replace(/[_-]+/g, " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
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
  const options = supported.map((value) => ({ value, label: thinkingLevelLabel(value) }));
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
  const level = normalizeThinkingLevel(key);
  return level ? thinkingLevelLabel(level) : undefined;
}

/** Model name plus a thinking word. No model name means no word. */
export function modelNameWithThinking(modelName: string, thinkingLevel: string | undefined): string {
  const word = visibleThinkingLevelLabel(thinkingLevel);
  if (!modelName || !word) {
    return modelName;
  }
  return `${modelName} ${word}`;
}
