import { describe, expect, it } from "bun:test";

import {
  addIntakeLabel,
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

  it("adds a typed label once and ignores blank input", () => {
    expect(addIntakeLabel(["bug"], "  needs-triage  ")).toEqual(["bug", "needs-triage"]);
    expect(addIntakeLabel(["bug"], "bug")).toEqual(["bug"]);
    expect(addIntakeLabel(["bug"], "   ")).toEqual(["bug"]);
  });

  it("clamps the confidence score", () => {
    const next = normalizeIntakeSourceSettings({
      ...DEFAULT_GITHUB_INTAKE_SETTINGS,
      confidencePct: 140.6,
    });

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

  it("round-trips GitHub issue events through the API shape", () => {
    const settings = intakeSettingsFromApi("GitHub issues", {
      newIssues: false,
      reopenedIssues: true,
      superplaneLabelAdded: true,
    });

    expect(settings.newIssues).toBe(false);
    expect(settings.reopenedIssues).toBe(true);
    expect(settings.superplaneLabelAdded).toBe(true);
    expect(intakeSettingsToApi(settings)).toMatchObject({
      newIssues: false,
      reopenedIssues: true,
      superplaneLabelAdded: true,
    });
  });

  it("keeps the new and re-opened toggles apart", () => {
    const settings = intakeSettingsFromApi("GitHub issues", {
      newIssues: true,
      reopenedIssues: false,
    });

    expect(settings.newIssues).toBe(true);
    expect(settings.reopenedIssues).toBe(false);
  });

  it("defaults the GitHub issue event toggles on when the API omits them", () => {
    const settings = intakeSettingsFromApi("GitHub issues", {});

    expect(settings.newIssues).toBe(true);
    expect(settings.reopenedIssues).toBe(true);
    expect(settings.superplaneLabelAdded).toBe(true);
  });
});
