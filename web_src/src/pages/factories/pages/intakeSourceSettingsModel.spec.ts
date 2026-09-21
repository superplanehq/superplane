import { describe, expect, it } from "bun:test";

import {
  addIntakeLabel,
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_SENTRY_INTAKE_SETTINGS,
  isIntakeSettingsTab,
  intakeSettingsTabs,
  intakeSettingsFromApi,
  intakeSettingsToApi,
  intakeSupportsPause,
  jiraCompletionSettingsToApi,
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

  it("round-trips the Jira completion column through the API shape", () => {
    const settings = intakeSettingsFromApi("Jira issues", {
      jiraMoveOnComplete: false,
      jiraCompletionColumn: "QA",
    });

    expect(settings.jiraMoveOnComplete).toBe(false);
    expect(settings.jiraCompletionColumn).toBe("QA");
    expect(intakeSettingsToApi(settings)).toMatchObject({
      jiraMoveOnComplete: false,
      jiraCompletionColumn: "",
    });
  });

  it("defaults the Jira completion move on when the API omits it", () => {
    const settings = intakeSettingsFromApi("Jira issues", {});

    expect(settings.jiraMoveOnComplete).toBe(true);
    expect(settings.jiraCompletionColumn).toBe("");
    expect(jiraCompletionSettingsToApi(settings)).toEqual({
      jiraMoveOnComplete: true,
      jiraCompletionColumn: "",
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

  it("round-trips Sentry events and levels through the API shape", () => {
    const settings = intakeSettingsFromApi("Sentry exceptions", {
      sentryNewIssues: false,
      sentryRegressedIssues: true,
      sentryAssignedIssues: true,
      sentryLevels: ["error", "unknown", "fatal"],
    });

    expect(settings.sentryNewIssues).toBe(false);
    expect(settings.sentryRegressedIssues).toBe(true);
    expect(settings.sentryAssignedIssues).toBe(true);
    expect(settings.sentryLevels).toEqual(["fatal", "error"]);
    expect(intakeSettingsToApi(settings)).toMatchObject({
      sentryNewIssues: false,
      sentryRegressedIssues: true,
      sentryAssignedIssues: true,
      sentryLevels: ["fatal", "error"],
    });
  });

  it("defaults Sentry events on when the API omits them", () => {
    const settings = intakeSettingsFromApi("Sentry exceptions", {});

    expect(settings.sentryNewIssues).toBe(DEFAULT_SENTRY_INTAKE_SETTINGS.sentryNewIssues);
    expect(settings.sentryRegressedIssues).toBe(DEFAULT_SENTRY_INTAKE_SETTINGS.sentryRegressedIssues);
    expect(settings.sentryAssignedIssues).toBe(DEFAULT_SENTRY_INTAKE_SETTINGS.sentryAssignedIssues);
    expect(settings.sentryLevels).toEqual([]);
  });

  it("offers pause for Sentry and Jira intakes only", () => {
    expect(intakeSupportsPause("sentry-exceptions")).toBe(true);
    expect(intakeSupportsPause("jira-issues")).toBe(true);
    expect(intakeSupportsPause("github-issues")).toBe(false);
    expect(intakeSupportsPause("productive-tasks")).toBe(false);
    expect(intakeSupportsPause("pagerduty-incidents")).toBe(false);
  });
});
