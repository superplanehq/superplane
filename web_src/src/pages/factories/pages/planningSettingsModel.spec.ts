import { describe, expect, it } from "bun:test";

import {
  DEFAULT_PLANNING_SETTINGS,
  factoryPlanningEnabled,
  factoryShowsClarity,
  factoryShowsConfidence,
  isPlanningSettingsTab,
  planningSettingsFromFactory,
  planningSettingsTabs,
  planningSettingsToApi,
} from "./planningSettingsModel";

describe("planningSettingsFromFactory", () => {
  it("defaults Planning on with both scores", () => {
    expect(planningSettingsFromFactory(undefined)).toEqual(DEFAULT_PLANNING_SETTINGS);
    expect(planningSettingsFromFactory({ id: "factory-1" })).toEqual(DEFAULT_PLANNING_SETTINGS);
  });

  it("reads stored Planning settings", () => {
    expect(
      planningSettingsFromFactory({
        id: "factory-1",
        planning: { enabled: false, clarity: true, confidence: false },
      }),
    ).toEqual({ enabled: false, clarity: true, confidence: false });
  });
});

describe("factory score visibility", () => {
  it("hides scores when Planning is off", () => {
    const factory = { id: "factory-1", planning: { enabled: false, clarity: true, confidence: true } };
    expect(factoryPlanningEnabled(factory)).toBe(false);
    expect(factoryShowsClarity(factory)).toBe(false);
    expect(factoryShowsConfidence(factory)).toBe(false);
  });

  it("hides one score when that toggle is off", () => {
    const factory = { id: "factory-1", planning: { enabled: true, clarity: false, confidence: true } };
    expect(factoryPlanningEnabled(factory)).toBe(true);
    expect(factoryShowsClarity(factory)).toBe(false);
    expect(factoryShowsConfidence(factory)).toBe(true);
  });
});

describe("planningSettingsTabs", () => {
  it("includes the agent tab when an agent node exists", () => {
    expect(planningSettingsTabs(true)).toEqual(["general", "agent", "automation"]);
    expect(planningSettingsTabs(false)).toEqual(["general", "automation"]);
  });
});

describe("isPlanningSettingsTab", () => {
  it("accepts the three settings tabs", () => {
    expect(isPlanningSettingsTab("general")).toBe(true);
    expect(isPlanningSettingsTab("agent")).toBe(true);
    expect(isPlanningSettingsTab("automation")).toBe(true);
    expect(isPlanningSettingsTab("runs")).toBe(false);
  });
});

describe("planningSettingsToApi", () => {
  it("sends the stored score flags when Planning is off", () => {
    expect(planningSettingsToApi({ enabled: false, clarity: true, confidence: false })).toEqual({
      enabled: false,
      clarity: true,
      confidence: false,
    });
  });
});
