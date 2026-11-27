import { describe, expect, it } from "bun:test";

import superplaneMark from "@/assets/superplane-mark.svg";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "../../lib/customAutomation";
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

  it("names a custom automation after itself even when its name says intake", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: undefined,
        createdBy: {
          automation: { appId: "app-alerts", appName: "Alert intake", templateId: CUSTOM_AUTOMATION_TEMPLATE_ID },
        },
      }),
    ).toEqual(expect.objectContaining({ kind: "automation", name: "Alert intake" }));
  });

  it("shows a custom automation link on an unknown host as a webhook, not a GitHub issue", () => {
    const automation = { appId: "app-requests", appName: "Request intake", templateId: CUSTOM_AUTOMATION_TEMPLATE_ID };
    const order = { ...DRAFT_WORK_ORDER, createdBy: { automation } };
    expect(
      splitRunSourceForOrder({ ...order, origin: { url: "https://requests.example.com/REQ-1", label: "REQ-1" } }),
    ).toEqual(
      expect.objectContaining({ kind: "intake", name: "Webhook", ticket: expect.objectContaining({ label: "REQ-1" }) }),
    );
    expect(
      splitRunSourceForOrder({ ...order, origin: { url: "https://github.com/acme/app/issues/7", label: "" } }),
    ).toEqual(expect.objectContaining({ name: "GitHub issues" }));
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
