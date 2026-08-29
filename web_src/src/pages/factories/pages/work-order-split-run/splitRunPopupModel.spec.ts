import { describe, expect, it } from "vitest";

import { factoryAppConfigurePath, factoryAppSplitRunPath } from "../../lib/factoryPagePaths";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import {
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  LINE_RUN_IMPLEMENT_NOTIFY_ID,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { LINE_BOARD_DONE_RECEIPTS_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import {
  collectSplitRunArtifacts,
  defaultSplitRunPopupTab,
  resolveSplitRunPopupArtifacts,
  SPLIT_RUN_PANE_GRID_CLASSNAME,
  splitRunAutomationRunHref,
  splitRunDescriptionMarkdown,
  splitRunLinkedArtifacts,
  splitRunLogTabDotClass,
  splitRunPhaseAutomationHref,
  splitRunPhaseRunHref,
  splitRunSourceDescription,
} from "./splitRunPopupModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

describe("splitRunPopupModel", () => {
  it("uses a 3/2 pane split for Description", () => {
    expect(SPLIT_RUN_PANE_GRID_CLASSNAME).toContain("minmax(0,3fr)_minmax(0,2fr)");
  });

  it("opens the automation run for the preferred phase, then the latest phase run", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER);
    const implementHref = getWorkOrderRunHref(
      FACTORIES_ORGANIZATION_ID,
      PRIMARY_FACTORY_KEY,
      "app-refund-implementer",
      LINE_RUN_IMPLEMENT_NOTIFY_ID,
      { orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number },
    );

    expect(
      splitRunAutomationRunHref({
        organizationId: FACTORIES_ORGANIZATION_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
        fixture,
        preferredPhaseId: "implementation-1",
      }),
    ).toBe(implementHref);
    expect(
      splitRunAutomationRunHref({
        organizationId: FACTORIES_ORGANIZATION_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
        fixture,
        preferredPhaseId: "pr-creation-2",
      }),
    ).toBe(implementHref);
    expect(
      splitRunAutomationRunHref({
        fixture,
        preferredPhaseId: "implementation-1",
      }),
    ).toBeNull();
    expect(
      splitRunAutomationRunHref({
        organizationId: FACTORIES_ORGANIZATION_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
      }),
    ).toBeNull();
  });

  it("opens the split-run page for a phase automation", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER);
    const prCreation = fixture.phases.find((phase) => phase.id === "pr-creation-2");
    expect(prCreation?.appId).toBe("app-pr-closure");

    expect(
      splitRunPhaseRunHref({
        organizationId: FACTORIES_ORGANIZATION_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
        phase: prCreation!,
      }),
    ).toBe(
      factoryAppSplitRunPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "app-pr-closure", {
        from: "work-order",
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
        canvas: "closure",
      }),
    );
    expect(
      splitRunPhaseAutomationHref({
        organizationId: FACTORIES_ORGANIZATION_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
        phase: prCreation!,
      }),
    ).toBe(
      factoryAppConfigurePath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "app-pr-closure", {
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
      }),
    );
    expect(splitRunPhaseRunHref({ phase: prCreation! })).toBeUndefined();
  });

  it("opens the description tab for drafts and done cards, and the log for later states", () => {
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER))).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER))).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER))).toBe("log");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER))).toBe("log");
  });

  it("maps log-tab dots to the line status colors", () => {
    expect(splitRunLogTabDotClass("running")).toContain("--status-running-dot");
    expect(splitRunLogTabDotClass("waiting")).toContain("--status-waiting-dot");
    expect(splitRunLogTabDotClass("failed")).toContain("--status-failed-dot");
    expect(splitRunLogTabDotClass("passed")).toContain("--status-completed-dot");
    expect(splitRunLogTabDotClass("pending")).toContain("--status-draft-dot");
  });

  it("prefers the saved work-order description on a live order", () => {
    expect(
      splitRunSourceDescription({
        workOrderDescription: "Saved on the task",
        artifactDescription: "Stale artifact body",
        preferWorkOrder: true,
      }),
    ).toBe("Saved on the task");
    expect(
      splitRunSourceDescription({
        workOrderDescription: "Saved on the task",
        artifactDescription: "Storybook artifact body",
      }),
    ).toBe("Storybook artifact body");
    expect(splitRunSourceDescription({ workOrderDescription: "  ", artifactDescription: "Artifact fallback" })).toBe(
      "Artifact fallback",
    );
  });

  it("reads details.md as the work-order description", () => {
    const fixture = splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]);
    const artifacts = collectSplitRunArtifacts(fixture);
    const description = splitRunDescriptionMarkdown(artifacts);

    expect(description).toContain("Webhook delivery stops after a transient provider error");
    expect(artifacts.some((artifact) => artifact.id?.endsWith("-plan"))).toBe(true);
    expect(splitRunLinkedArtifacts(artifacts).some((artifact) => artifact.id?.endsWith("-details"))).toBe(false);
    expect(splitRunLinkedArtifacts(artifacts).some((artifact) => artifact.id?.endsWith("-plan"))).toBe(true);
  });

  it("uses live artifacts for a real task and fixture artifacts in Storybook", () => {
    const fixtureArtifacts = collectSplitRunArtifacts(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER));
    const liveArtifacts = [
      {
        id: "art-live-link",
        type: "TYPE_LINK" as const,
        data: { title: "Preview", url: "https://preview.example.com/88" },
      },
    ];

    expect(resolveSplitRunPopupArtifacts({ fixtureArtifacts, liveArtifacts, useLive: true })).toEqual(liveArtifacts);
    expect(resolveSplitRunPopupArtifacts({ fixtureArtifacts, liveArtifacts, useLive: false })).toEqual(
      fixtureArtifacts,
    );
  });

  it("lists description artifacts oldest first", () => {
    const artifacts = splitRunLinkedArtifacts([
      {
        id: "newer",
        type: "TYPE_LINK",
        createdAt: "2026-08-25T12:00:00.000Z",
        data: { title: "Preview", url: "https://preview.example.com/2" },
      },
      {
        id: "older",
        type: "TYPE_BRANCH",
        createdAt: "2026-08-25T10:00:00.000Z",
        data: { name: "feature/a" },
      },
      {
        id: "description",
        type: "TYPE_MARKDOWN",
        createdAt: "2026-08-25T09:00:00.000Z",
        data: { name: "description.md" },
      },
      { id: "undated", type: "TYPE_LINK", data: { title: "late" } },
    ]);

    expect(artifacts.map((artifact) => artifact.id)).toEqual(["older", "newer", "undated"]);
  });
});
