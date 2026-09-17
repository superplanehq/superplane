import { describe, expect, it } from "vitest";

import { sentryIntakePreviewCaption, sentryIntakePreviewRows } from "./sentryIntakePreview";
import { SENTRY_INTAKE_SEED_SIZE, SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

describe("sentryIntakePreviewRows", () => {
  it("uses example issues when the catalog is empty", () => {
    const rows = sentryIntakePreviewRows([]);
    expect(rows).toHaveLength(SENTRY_INTAKE_SEED_SIZE);
    expect(rows[0]?.title).toBe(SENTRY_INTAKE_SETUP_COPY.exampleIssues[0]);
  });

  it("keeps at most 10 catalog issues", () => {
    const rows = sentryIntakePreviewRows(["One", "Two", "Three"]);
    expect(rows.map((row) => row.title)).toEqual(["One", "Two", "Three"]);

    const overflow = sentryIntakePreviewRows(Array.from({ length: 15 }, (_, index) => `Issue ${index + 1}`));
    expect(overflow).toHaveLength(SENTRY_INTAKE_SEED_SIZE);
    expect(overflow.at(-1)?.title).toBe("Issue 10");
  });
});

describe("sentryIntakePreviewCaption", () => {
  it("asks for a project before a project is selected", () => {
    expect(sentryIntakePreviewCaption(false)).toBe(SENTRY_INTAKE_SETUP_COPY.wizardPreviewCaptionNoProject);
  });

  it("states the 10-issue import and the listener after a project is selected", () => {
    expect(sentryIntakePreviewCaption(true)).toBe(SENTRY_INTAKE_SETUP_COPY.wizardPreviewCaption);
  });
});
