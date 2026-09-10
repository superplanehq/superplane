import { describe, expect, it } from "vitest";

import {
  checksPreviewAttemptsLabel,
  checksPreviewReadingLabel,
  checksPreviewRows,
  checksPreviewToolLabel,
  checksPreviewToolOutcome,
  checksSetupPreviewCaption,
} from "./checksPRFeedbackPreview";
import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";

describe("checksPreviewRows", () => {
  it("puts selected checks first and marks the rest as ignored", () => {
    expect(checksPreviewRows(["lint", "e2e", "build"], ["e2e"])).toEqual([
      { name: "e2e", selected: true },
      { name: "lint", selected: false },
      { name: "build", selected: false },
    ]);
  });

  it("keeps catalog order when every check is selected", () => {
    expect(checksPreviewRows(["lint", "e2e"], ["lint", "e2e"])).toEqual([
      { name: "lint", selected: true },
      { name: "e2e", selected: true },
    ]);
  });

  it("shows catalog checks as ignored when none are selected", () => {
    expect(checksPreviewRows(["lint", "e2e"], [])).toEqual([
      { name: "lint", selected: false },
      { name: "e2e", selected: false },
    ]);
  });

  it("limits the preview list", () => {
    expect(checksPreviewRows(["a", "b", "c", "d", "e"], ["e", "d"], 3)).toEqual([
      { name: "e", selected: true },
      { name: "d", selected: true },
      { name: "a", selected: false },
    ]);
  });
});

describe("checksPreviewAttemptsLabel", () => {
  it("uses a singular label for one attempt", () => {
    expect(checksPreviewAttemptsLabel(1)).toBe(PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksAttemptsOne);
  });

  it("uses a plural label for more than one attempt", () => {
    expect(checksPreviewAttemptsLabel(3)).toBe("Stops after 3 attempts.");
  });

  it("hides the label when the attempt count is not a valid integer", () => {
    expect(checksPreviewAttemptsLabel(0)).toBe("");
    expect(checksPreviewAttemptsLabel(2.5)).toBe("");
  });
});

describe("checksPreviewToolOutcome", () => {
  const connected = [{ metadata: { id: "int-cci", integrationName: "circleci" } }];

  it("marks suggested tools as granted when a matching integration is selected", () => {
    expect(
      checksPreviewToolOutcome({
        toolsAccess: "suggested",
        suggestedNames: ["circleci"],
        selectedIds: ["int-cci"],
        connected,
      }),
    ).toEqual({ kind: "granted", labels: ["CircleCI"] });
  });

  it("marks suggested tools as missing access when none are selected", () => {
    expect(
      checksPreviewToolOutcome({
        toolsAccess: "suggested",
        suggestedNames: ["circleci"],
        selectedIds: [],
        connected,
      }),
    ).toEqual({ kind: "no-access", labels: ["CircleCI"] });
  });

  it("describes GitHub Actions without extra tools", () => {
    expect(
      checksPreviewToolOutcome({
        toolsAccess: "github-actions",
        suggestedNames: [],
        selectedIds: [],
        connected: [],
      }),
    ).toEqual({ kind: "github-actions", labels: [PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGitHub] });
  });

  it("describes checks that need no additional access", () => {
    expect(
      checksPreviewToolOutcome({
        toolsAccess: "none",
        suggestedNames: [],
        selectedIds: [],
        connected: [],
      }),
    ).toEqual({ kind: "none", labels: [] });
  });
});

describe("checksPreviewReadingLabel", () => {
  it("names one tool", () => {
    expect(checksPreviewReadingLabel(["CircleCI"])).toBe("Reading CircleCI logs.");
  });

  it("joins two tools with and", () => {
    expect(checksPreviewReadingLabel(["CircleCI", "Semaphore"])).toBe("Reading CircleCI and Semaphore logs.");
  });
});

describe("checksPreviewToolLabel", () => {
  it("keeps known CI display names", () => {
    expect(checksPreviewToolLabel("circleci")).toBe("CircleCI");
    expect(checksPreviewToolLabel("semaphore")).toBe("Semaphore");
  });
});

describe("checksSetupPreviewCaption", () => {
  it("describes the checks step", () => {
    expect(checksSetupPreviewCaption({ step: "checks", catalogEmpty: true, selectedCount: 0 })).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksEmptyCaption,
    );
    expect(checksSetupPreviewCaption({ step: "checks", catalogEmpty: false, selectedCount: 0 })).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksNoneCaption,
    );
    expect(checksSetupPreviewCaption({ step: "checks", catalogEmpty: false, selectedCount: 2 })).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksSelectedCaption,
    );
  });

  it("describes the tools step from the access outcome", () => {
    expect(
      checksSetupPreviewCaption({
        step: "tools",
        catalogEmpty: false,
        selectedCount: 1,
        toolOutcome: { kind: "granted", labels: ["CircleCI"] },
      }),
    ).toBe(PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGrantedCaption);
    expect(
      checksSetupPreviewCaption({
        step: "tools",
        catalogEmpty: false,
        selectedCount: 1,
        toolOutcome: { kind: "no-access", labels: ["CircleCI"] },
      }),
    ).toBe(PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsNoAccessCaption);
    expect(
      checksSetupPreviewCaption({
        step: "tools",
        catalogEmpty: false,
        selectedCount: 1,
        toolOutcome: { kind: "github-actions", labels: ["GitHub Actions"] },
      }),
    ).toBe(PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGitHubCaption);
    expect(
      checksSetupPreviewCaption({
        step: "tools",
        catalogEmpty: false,
        selectedCount: 1,
        toolOutcome: { kind: "none", labels: [] },
      }),
    ).toBe(PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsNoneCaption);
  });
});
