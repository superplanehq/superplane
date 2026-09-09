import { describe, expect, it } from "vitest";

import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  isIntakeSettingsTab,
  intakeSettingsTabs,
  intakeSettingsFromApi,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  toggleIntakeLabel,
} from "./intakeSourceSettingsModel";

describe("intakeSourceSettingsModel", () => {
  it("adds and removes a label", () => {
    expect(toggleIntakeLabel([], "bug")).toEqual(["bug"]);
    expect(toggleIntakeLabel(["bug", "enhancement"], "bug")).toEqual(["enhancement"]);
  });

  it("keeps a default name when the draft name is empty", () => {
    const next = normalizeIntakeSourceSettings({
      ...DEFAULT_GITHUB_INTAKE_SETTINGS,
      name: "   ",
      confidencePct: 140.6,
    });

    expect(next.name).toBe("GitHub issues");
    expect(next.confidencePct).toBe(100);
  });

  it("accepts the intake settings tabs", () => {
    expect(isIntakeSettingsTab("automation")).toBe(true);
    expect(isIntakeSettingsTab("runs")).toBe(false);
    expect(isIntakeSettingsTab("agent")).toBe(true);
    expect(isIntakeSettingsTab("general")).toBe(true);
    expect(isIntakeSettingsTab("listen")).toBe(false);
    expect(intakeSettingsTabs(false)).toEqual(["general", "automation"]);
    expect(intakeSettingsTabs(true)).toEqual(["general", "agent", "automation"]);
  });

  it("defaults the authors filter to off", () => {
    expect(DEFAULT_GITHUB_INTAKE_SETTINGS.authorsWithAccess).toBe(false);
    expect(intakeSettingsFromApi("GitHub issues", undefined).authorsWithAccess).toBe(false);
  });

  it("round-trips the authors filter through the API shape", () => {
    const on = intakeSettingsFromApi("GitHub issues", { authorsWithAccess: true });
    expect(on.authorsWithAccess).toBe(true);
    expect(intakeSettingsToApi(on).authorsWithAccess).toBe(true);

    const off = intakeSettingsFromApi("GitHub issues", { authorsWithAccess: false });
    expect(off.authorsWithAccess).toBe(false);
    expect(intakeSettingsToApi(off).authorsWithAccess).toBe(false);
  });
});
