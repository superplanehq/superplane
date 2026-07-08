import type { RunInspectorNodeSection } from "./runNodeDetailModel";

export const ACCORDION_STORAGE_KEY = "superplane.runInspector.internalAccordions";

export type InternalAccordionKey = "input" | "runtime" | "output";
export type InternalAccordionPreferences = Record<InternalAccordionKey, boolean>;
export type TimelineStepType = "input" | "runtime" | "output";

export type StatusPill = {
  dotClassName: string;
  label: string;
  tone?: "default" | "error";
};

export const defaultAccordionPreferences: InternalAccordionPreferences = {
  input: false,
  runtime: false,
  output: false,
};

export function buildTimelineItems(section: RunInspectorNodeSection, hasRuntimeConfig: boolean) {
  const items: Array<{
    value: TimelineStepType;
  }> = [];

  if (!section.isTrigger) {
    items.push({ value: "input" });
  }

  if (hasRuntimeConfig) {
    items.push({ value: "runtime" });
  }

  items.push({ value: "output" });

  return items;
}

export function readAccordionPreferences(): InternalAccordionPreferences {
  const storedValue = localStorage.getItem(ACCORDION_STORAGE_KEY);
  if (!storedValue) return defaultAccordionPreferences;

  try {
    const parsed = JSON.parse(storedValue) as Partial<InternalAccordionPreferences>;
    return {
      input: parsed.input ?? defaultAccordionPreferences.input,
      runtime: parsed.runtime ?? defaultAccordionPreferences.runtime,
      output: parsed.output ?? defaultAccordionPreferences.output,
    };
  } catch {
    return defaultAccordionPreferences;
  }
}
