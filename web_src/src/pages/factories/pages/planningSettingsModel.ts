import type { FactoriesFactory } from "@/api-client";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";

export { PLANNING_SETTINGS_COPY };

export type PlanningSettingsTab = "general" | "agent" | "automation";

export type PlanningDraftSettings = {
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
  autoStart: boolean;
  autoStartLine: string;
};

/** Mirrors DefaultFactoryPlanning on the server: Clarity and automatic start are opt-in. */
export const DEFAULT_PLANNING_SETTINGS: PlanningDraftSettings = {
  enabled: true,
  clarity: false,
  confidence: true,
  autoStart: false,
  autoStartLine: "",
};

export function isPlanningSettingsTab(value: string | null | undefined): value is PlanningSettingsTab {
  return value === "general" || value === "agent" || value === "automation";
}

export function planningSettingsTabs(hasAgent: boolean): PlanningSettingsTab[] {
  return hasAgent ? ["general", "agent", "automation"] : ["general", "automation"];
}

export function planningSettingsFromFactory(factory?: FactoriesFactory | null): PlanningDraftSettings {
  const planning = factory?.planning;
  return {
    ...DEFAULT_PLANNING_SETTINGS,
    ...definedPlanningFields({
      enabled: planning?.enabled,
      clarity: planning?.clarity,
      confidence: planning?.confidence,
      autoStart: planning?.autoStart,
      autoStartLine: planning?.autoStartLine,
    }),
  };
}

function definedPlanningFields(fields: Partial<PlanningDraftSettings>): Partial<PlanningDraftSettings> {
  return Object.fromEntries(
    Object.entries(fields).filter(([, value]) => value !== undefined),
  ) as Partial<PlanningDraftSettings>;
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
    autoStart: settings.autoStart,
    autoStartLine: settings.autoStartLine,
    setupCompleted: true,
  };
}
