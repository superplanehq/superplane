import type { RunInspectorNodeSection } from "./runNodeDetailModel";

export const ACCORDION_STORAGE_KEY = "superplane.runInspector.internalAccordions";

export type InternalAccordionKey = "input" | "runtime" | "logs" | "output";
export type InternalAccordionPreferences = Record<InternalAccordionKey, boolean>;
export type TimelineStepType = "input" | "runtime" | "logs" | "output";

export type StatusPill = {
  dotClassName: string;
  label: string;
  tone?: "default" | "error";
};

export const defaultAccordionPreferences: InternalAccordionPreferences = {
  input: false,
  runtime: false,
  logs: false,
  output: false,
};

export function buildTimelineItems(
  section: RunInspectorNodeSection,
  hasRuntimeConfig: boolean,
  hasRunnerLogs = false,
): Array<{ value: TimelineStepType }> {
  const items: Array<{
    value: TimelineStepType;
  }> = [];

  if (section.isQueued) {
    return [{ value: "input" }];
  }

  if (!section.isTrigger) {
    items.push({ value: "input" });
  }

  if (hasRuntimeConfig) {
    items.push({ value: "runtime" });
  }

  if (hasRunnerLogs) {
    items.push({ value: "logs" });
  }

  if (section.outputSections.length > 0 || section.errorMessage) {
    items.push({ value: "output" });
  }

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
      logs: parsed.logs ?? defaultAccordionPreferences.logs,
      output: parsed.output ?? defaultAccordionPreferences.output,
    };
  } catch {
    return defaultAccordionPreferences;
  }
}
