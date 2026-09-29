import type { FactoriesFactory } from "@/api-client";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";

export { PLANNING_SETTINGS_COPY };

export type PlanningSettingsTab = "general" | "agent" | "automation";

export type PlanningAutoStartLine = {
  id: string;
  name: string;
};

export type PlanningDraftSettings = {
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
  autoStartLineId: string;
};

/** Mirrors DefaultFactoryPlanning on the server: Clarity is opt-in. Auto-start is off. */
export const DEFAULT_PLANNING_SETTINGS: PlanningDraftSettings = {
  enabled: true,
  clarity: false,
  confidence: true,
  autoStartLineId: "",
};

export function isPlanningSettingsTab(value: string | null | undefined): value is PlanningSettingsTab {
  return value === "general" || value === "agent" || value === "automation";
}

export function planningSettingsTabs(hasAgent: boolean): PlanningSettingsTab[] {
  return hasAgent ? ["general", "agent", "automation"] : ["general", "automation"];
}

export function planningAutoStartLines(factory?: FactoriesFactory | null): PlanningAutoStartLine[] {
  return (factory?.lines ?? []).flatMap((line) => {
    const id = line.id?.trim() ?? "";
    const name = line.name?.trim() ?? "";
    return id && name ? [{ id, name }] : [];
  });
}

export function planningSettingsFromFactory(factory?: FactoriesFactory | null): PlanningDraftSettings {
  const stored = factory?.planning?.autoStartLineId ?? "";
  const lines = factory?.lines;
  const autoStartLineId = !lines || lines.some((line) => line.id === stored) ? stored : "";
  return {
    enabled: factory?.planning?.enabled ?? DEFAULT_PLANNING_SETTINGS.enabled,
    clarity: factory?.planning?.clarity ?? DEFAULT_PLANNING_SETTINGS.clarity,
    confidence: factory?.planning?.confidence ?? DEFAULT_PLANNING_SETTINGS.confidence,
    autoStartLineId,
  };
}

export function factoryPlanningEnabled(factory?: FactoriesFactory | null): boolean {
  return planningSettingsFromFactory(factory).enabled;
}

export function factoryShowsClarity(factory?: FactoriesFactory | null): boolean {
  const settings = planningSettingsFromFactory(factory);
  return settings.enabled && settings.clarity;
}

export function factoryShowsConfidence(factory?: FactoriesFactory | null): boolean {
  const settings = planningSettingsFromFactory(factory);
  return settings.enabled && settings.confidence;
}

export type PlanningApiSettings = PlanningDraftSettings & { setupCompleted: true };

export function planningSettingsToApi(settings: PlanningDraftSettings): PlanningApiSettings {
  return {
    enabled: settings.enabled,
    clarity: settings.clarity,
    confidence: settings.confidence,
    autoStartLineId: settings.autoStartLineId,
    setupCompleted: true,
  };
}
