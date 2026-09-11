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
  collectSplitRunPullRequests,
  defaultSplitRunPopupTab,
  resolveSplitRunPopupArtifacts,
  resolveSplitRunPopupPullRequests,
  splitRunAutomationRunHref,
  splitRunDescriptionMarkdown,
  splitRunIntentDocument,
  splitRunLinkedArtifacts,
  splitRunPhaseAutomationHref,
  splitRunPhaseRunHref,
  splitRunSourceDescription,
} from "./splitRunPopupModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

describe("splitRunPopupModel", () => {
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
        from: "task",
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
    expect(
      defaultSplitRunPopupTab(
        splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
          analysisRuns: [
            {
              canvasId: "canvas-backlog",
              workOrderId: DRAFT_WORK_ORDER.id ?? "",
              run: {
                id: "run-analysis",
                canvasId: "canvas-backlog",
                state: "STATE_STARTED",
                createdAt: "2026-08-28T12:00:00Z",
                updatedAt: "2026-08-28T12:00:00Z",
              },
            },
          ],
        }),
      ),
    ).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER))).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER))).toBe("description");
    expect(defaultSplitRunPopupTab(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER))).toBe("log");
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
    expect(splitRunIntentDocument({ artifacts, description }).summary).toBeTruthy();
    expect(splitRunIntentDocument({ artifacts, description }).plan).toContain("##");
  });

  it("prefers spec.md over intent.md", () => {
    const artifacts = [
      {
        id: "art-intent",
        type: "TYPE_MARKDOWN" as const,
        data: { name: "intent.md", body: "# Old intent\n\n## Executive summary\n\nOld summary.\n" },
      },
      {
        id: "art-spec",
        type: "TYPE_MARKDOWN" as const,
        data: { name: "spec.md", body: "# New spec\n\n## Executive summary\n\nNew summary.\n" },
      },
    ];

    expect(splitRunIntentDocument({ artifacts, description: "Webhook timeouts." }).title).toBe("New spec");
    expect(splitRunIntentDocument({ artifacts, description: "Webhook timeouts." }).summary).toBe("New summary.");
    expect(splitRunLinkedArtifacts(artifacts)).toEqual([]);
  });

  it("keeps intent.md out of the Artifacts list", () => {
    const artifacts = [
      {
        id: "art-intent",
        type: "TYPE_MARKDOWN" as const,
        data: { name: "intent.md", body: "## Executive summary\n\nA retry loop." },
      },
      {
        id: "art-plan",
        type: "TYPE_MARKDOWN" as const,
        data: { name: "plan.md", body: "Add a retry." },
      },
    ];

    expect(splitRunLinkedArtifacts(artifacts).map((artifact) => artifact.id)).toEqual(["art-plan"]);
    expect(splitRunIntentDocument({ artifacts, description: "Webhook timeouts." }).summary).toBe("A retry loop.");
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

  it("collects unique pull requests from fixture streams", () => {
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER);
    const pullRequests = collectSplitRunPullRequests(fixture);

    expect(pullRequests).toEqual([
      expect.objectContaining({
        number: "510",
        url: "https://github.com/example/ledger/pull/510",
        state: "STATE_OPEN",
      }),
    ]);
  });

  it("uses live pull requests for a real task and fixture pull requests in Storybook", () => {
    const fixturePullRequests = collectSplitRunPullRequests(
      splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER),
    );
    const livePullRequests = [
      {
        id: "pr-live",
        number: "99",
        url: "https://github.com/example/ledger/pull/99",
        state: "STATE_OPEN" as const,
      },
    ];

    expect(resolveSplitRunPopupPullRequests({ fixturePullRequests, livePullRequests, useLive: true })).toEqual(
      livePullRequests,
    );
    expect(resolveSplitRunPopupPullRequests({ fixturePullRequests, livePullRequests, useLive: false })).toEqual(
      fixturePullRequests,
    );
    expect(collectSplitRunPullRequests(splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]))).toEqual([]);
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
