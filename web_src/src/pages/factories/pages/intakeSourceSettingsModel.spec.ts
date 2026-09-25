import { describe, expect, it } from "bun:test";

import {
  addIntakeLabel,
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_SENTRY_INTAKE_SETTINGS,
  isIntakeSettingsTab,
  intakeSettingsTabs,
  intakeSettingsFromApi,
  intakeSettingsToApi,
  intakeSupportsDelete,
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

  it("round-trips Sentry new-issue and level fields and turns off hidden triggers", () => {
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
      sentryRegressedIssues: false,
      sentryAssignedIssues: false,
      sentryLevels: ["fatal", "error"],
    });
  });

  it("defaults omitted Sentry event toggles from the Sentry intake defaults", () => {
    const settings = intakeSettingsFromApi("Sentry exceptions", {});

    expect(settings.sentryNewIssues).toBe(DEFAULT_SENTRY_INTAKE_SETTINGS.sentryNewIssues);
    expect(settings.sentryRegressedIssues).toBe(DEFAULT_SENTRY_INTAKE_SETTINGS.sentryRegressedIssues);
    expect(settings.sentryAssignedIssues).toBe(DEFAULT_SENTRY_INTAKE_SETTINGS.sentryAssignedIssues);
    expect(settings.sentryLevels).toEqual([]);
  });

  it("defaults excludeKeyTasks on when the API omits it", () => {
    const settings = intakeSettingsFromApi("Productive.io tasks", {});

    expect(settings.excludeKeyTasks).toBe(true);
    expect(intakeSettingsToApi(settings).excludeKeyTasks).toBe(true);
  });

  it("round-trips excludeKeyTasks through the API shape", () => {
    const settings = intakeSettingsFromApi("Productive.io tasks", { excludeKeyTasks: false });

    expect(settings.excludeKeyTasks).toBe(false);
    expect(intakeSettingsToApi(settings).excludeKeyTasks).toBe(false);
  });

  it("round-trips selected Productive.io task lists and drops blanks", () => {
    const settings = intakeSettingsFromApi("Productive.io tasks", {
      taskListIds: [" list-a ", "list-a", "", "list-b"],
    });

    expect(settings.taskListIds).toEqual(["list-a", "list-b"]);
    expect(intakeSettingsToApi(settings).taskListIds).toEqual(["list-a", "list-b"]);
    expect(intakeSettingsFromApi("Productive.io tasks", {}).taskListIds).toEqual([]);
  });

  it("offers delete for GitHub, Sentry, Jira, and Productive.io intakes", () => {
    expect(intakeSupportsDelete("github-issues")).toBe(true);
    expect(intakeSupportsDelete("sentry-exceptions")).toBe(true);
    expect(intakeSupportsDelete("jira-issues")).toBe(true);
    expect(intakeSupportsDelete("productive-tasks")).toBe(true);
    expect(intakeSupportsDelete("pagerduty-incidents")).toBe(false);
  });
});
