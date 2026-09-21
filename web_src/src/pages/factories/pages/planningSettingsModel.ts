import type { FactoriesFactory } from "@/api-client";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";

export { PLANNING_SETTINGS_COPY };

export type PlanningSettingsTab = "general" | "agent" | "automation";

export type PlanningDraftSettings = {
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
};

export const DEFAULT_PLANNING_SETTINGS: PlanningDraftSettings = {
  enabled: true,
  clarity: true,
  confidence: true,
};

export function isPlanningSettingsTab(value: string | null | undefined): value is PlanningSettingsTab {
  return value === "general" || value === "agent" || value === "automation";
}

export function planningSettingsTabs(hasAgent: boolean): PlanningSettingsTab[] {
  return hasAgent ? ["general", "agent", "automation"] : ["general", "automation"];
}

export function planningSettingsFromFactory(factory?: FactoriesFactory | null): PlanningDraftSettings {
  return {
    enabled: factory?.planning?.enabled ?? true,
    clarity: factory?.planning?.clarity ?? true,
    confidence: factory?.planning?.confidence ?? true,
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

export function factoryPlanningSetupCompleted(factory?: FactoriesFactory | null): boolean {
  return factory?.planning?.setupCompleted === true;
}

export type PlanningApiSettings = PlanningDraftSettings & { setupCompleted: true };

export function planningSettingsToApi(settings: PlanningDraftSettings): PlanningApiSettings {
  return {
    enabled: settings.enabled,
    clarity: settings.clarity,
    confidence: settings.confidence,
    setupCompleted: true,
  };
}
