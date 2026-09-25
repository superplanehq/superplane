import githubIcon from "@/assets/icons/integrations/github.svg";

import { LINE_INTAKE_SOURCES } from "../pages/lineIntakeModel";
import { availablePRFeedbackSources } from "../pages/prFeedbackSettingsModel";
import type {
  ColumnAutomation,
  ColumnAutomationCatalogEntry,
  ColumnAutomationKind,
  ColumnKey,
} from "./columnAutomations";

export const ANALYSIS_CATALOG_ID = "analysis";
export const AGENT_STEP_CATALOG_ID = "agent-step";
export const CUSTOM_CATALOG_ID = "custom";
export const PR_CLOSURE_CATALOG_ID = "pr-closure";

export const ANALYSIS_ENTRY: ColumnAutomationCatalogEntry = {
  id: ANALYSIS_CATALOG_ID,
  kind: "analysis",
  name: "Task analysis",
  description: "Plan a task when it enters Backlog.",
  trigger: "On task in Backlog",
  action: "Plan the task",
  iconSrc: "",
  iconAlt: "",
  unique: true,
};

export const AGENT_STEP_ENTRY: ColumnAutomationCatalogEntry = {
  id: AGENT_STEP_CATALOG_ID,
  kind: "agent-step",
  name: "Run an agent",
  description: "Start an agent when a task enters this column.",
  trigger: "On task in this column",
  action: "Run the agent",
  iconSrc: "",
  iconAlt: "",
  unique: false,
};

export const CUSTOM_ENTRY: ColumnAutomationCatalogEntry = {
  id: CUSTOM_CATALOG_ID,
  kind: "custom",
  name: "Custom automation",
  description: "Add a blank automation to this column.",
  trigger: "On task in this column",
  action: "Run the automation",
  iconSrc: "",
  iconAlt: "",
  unique: false,
};

export const EVENT_CUSTOM_ENTRY: ColumnAutomationCatalogEntry = {
  id: CUSTOM_CATALOG_ID,
  kind: "custom",
  name: "Custom automation",
  description: "Add a blank automation. Choose a trigger and the steps to run.",
  trigger: "On a trigger you choose",
  action: "Run the automation",
  iconSrc: "",
  iconAlt: "",
  unique: false,
};

export const PR_CLOSURE_ENTRY: ColumnAutomationCatalogEntry = {
  id: PR_CLOSURE_CATALOG_ID,
  kind: "pr-closure",
  name: "Pull request closure",
  description: "Complete the task when the pull request merges or closes.",
  trigger: "On pull request merged or closed",
  action: "Complete the task",
  iconSrc: githubIcon,
  iconAlt: "GitHub",
  unique: true,
};

export function catalogForColumn(key: ColumnKey, options?: { allowCustom?: boolean }): ColumnAutomationCatalogEntry[] {
  const allowCustom = options?.allowCustom === true;
  if (key === "backlog") {
    return [
      ...LINE_INTAKE_SOURCES.map((source) => ({
        id: source.id,
        kind: "intake" as const,
        name: source.name,
        description: source.description,
        trigger: source.listen.label,
        action: source.accept.label,
        iconSrc: source.iconSrc,
        iconAlt: source.iconAlt,
        unique: true,
      })),
      ANALYSIS_ENTRY,
    ];
  }
  if (key === "verify") {
    return [
      ...availablePRFeedbackSources().map((source) => {
        const sentence = prFeedbackSentence(source.id);
        return {
          id: source.id,
          kind: sentence.kind,
          name: source.name,
          description: source.description,
          trigger: sentence.trigger,
          action: sentence.action,
          iconSrc: source.iconSrc,
          iconAlt: source.iconAlt,
          unique: true,
        };
      }),
      ...(allowCustom ? [EVENT_CUSTOM_ENTRY] : []),
    ];
  }
  if (key === "done") {
    return allowCustom ? [PR_CLOSURE_ENTRY, EVENT_CUSTOM_ENTRY] : [PR_CLOSURE_ENTRY];
  }
  return allowCustom ? [AGENT_STEP_ENTRY, CUSTOM_ENTRY] : [AGENT_STEP_ENTRY];
}

export function takenCatalogIds(automations: ColumnAutomation[], catalog: ColumnAutomationCatalogEntry[]): string[] {
  const present = new Set(automations.map((automation) => automation.catalogId));
  return catalog.filter((entry) => entry.unique && present.has(entry.id)).map((entry) => entry.id);
}

/** True when every unique catalog type is taken and custom is the only remaining choice. */
export function onlyCustomCatalogRemains(
  catalog: ColumnAutomationCatalogEntry[],
  takenIds: readonly string[],
): boolean {
  const remaining = catalog.filter((entry) => !takenIds.includes(entry.id));
  return remaining.length === 1 && remaining[0]?.kind === "custom";
}

export function prFeedbackSentence(sourceId: string): { kind: ColumnAutomationKind; trigger: string; action: string } {
  if (sourceId === "checks") {
    return { kind: "pr-checks", trigger: "On failing pull request check", action: "Fix the checks" };
  }
  return { kind: "pr-discussion", trigger: "On pull request comment", action: "Address the feedback" };
}
