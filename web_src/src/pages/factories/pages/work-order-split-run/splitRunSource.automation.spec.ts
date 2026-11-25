import { describe, expect, it } from "bun:test";

import superplaneMark from "@/assets/superplane-mark.svg";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { splitRunSourceForOrder } from "./splitRunSource";

describe("splitRunSourceForOrder automations", () => {
  // A task created by a run of an automation such as ESLint AutoFix names that
  // automation. It does not read as a GitHub issue.
  it("names an automation that is not an intake after the automation", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: undefined,
        createdBy: { automation: { appId: "canvas-eslint", appName: "ESLint AutoFix" } },
      }),
    ).toEqual(
      expect.objectContaining({
        kind: "automation",
        name: "ESLint AutoFix",
        iconAlt: "SuperPlane",
        iconSrc: superplaneMark,
        addedBy: { kind: "automation", name: "ESLint AutoFix" },
      }),
    );
  });

  it("keeps GitHub issues for an ingest automation without a ticket type", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: undefined,
        createdBy: { automation: { appId: "app-issue-intake", appName: "Issue intake" } },
      }),
    ).toEqual(expect.objectContaining({ kind: "intake", name: "GitHub issues" }));
  });

  it("links a task to its creating run and automation", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: undefined,
        sourceRunId: "run-eslint-1",
        createdBy: { automation: { appId: "canvas-eslint", appName: "ESLint AutoFix" } },
      }),
    ).toEqual(expect.objectContaining({ run: { appId: "canvas-eslint", runId: "run-eslint-1" } }));
  });

  it("omits the run link when the task has no source run", () => {
    const source = splitRunSourceForOrder({
      ...DRAFT_WORK_ORDER,
      origin: undefined,
      sourceRunId: undefined,
      createdBy: { automation: { appId: "canvas-eslint", appName: "ESLint AutoFix" } },
    });
    expect(source).not.toHaveProperty("run");
  });
});
