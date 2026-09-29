import { describe, expect, it } from "bun:test";

import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import { buildLinePhaseBoard } from "./linePhaseRuns";
import {
  applyOrderColumnView,
  applyPhaseColumnView,
  columnViewPageLoads,
  columnViewReadiness,
  defaultLineColumnView,
  withLineColumnAge,
  withLineColumnMinimum,
  withLineColumnNoScore,
  withLineColumnSort,
  type LineColumnViewChoice,
} from "./lineColumnView";

const NOW = new Date("2026-09-29T12:00:00.000Z");
const LINE_ID = "line-1";

const LINE: FactoriesFactoryLine = {
  id: LINE_ID,
  steps: [
    { type: "runApp", app: { app: "app-implement", entrypoint: "start" } },
    { type: "runApp", app: { app: "app-verify", entrypoint: "start" } },
  ],
};

function choice(patch: Partial<LineColumnViewChoice> = {}): LineColumnViewChoice {
  return { ...defaultLineColumnView(), ...patch };
}

function phaseOrder(input: {
  id: string;
  title: string;
  stepIndex: number;
  appId: string;
  createdAt: string;
  executionUpdatedAt: string;
  confidence?: number;
  originUrl?: string;
}): FactoriesWorkOrder {
  return {
    id: input.id,
    title: input.title,
    state: "STATE_OPEN",
    createdAt: input.createdAt,
    updatedAt: input.createdAt,
    origin: input.originUrl ? { url: input.originUrl, label: input.title } : undefined,
    checkScores: input.confidence == null ? [] : [{ name: "Confidence score", score: input.confidence, maxScore: 5 }],
    lineDispatches: [
      {
        id: `dispatch-${input.id}`,
        line: { id: LINE_ID },
        createdAt: input.executionUpdatedAt,
        stepExecutions: [
          {
            id: `exec-${input.id}`,
            stepIndex: input.stepIndex,
            state: "STATE_STARTED",
            createdAt: input.executionUpdatedAt,
            updatedAt: input.executionUpdatedAt,
            run: { id: `run-${input.id}`, appId: input.appId },
          },
        ],
      },
    ],
  } as FactoriesWorkOrder;
}

function placedPhase(stepIndex: number, orders: FactoriesWorkOrder[]) {
  return buildLinePhaseBoard(LINE, orders, [
    { id: "app-implement", name: "Implement" },
    { id: "app-verify", name: "Verify" },
  ])[stepIndex].runs;
}

describe("applyPhaseColumnView", () => {
  const lowScore = phaseOrder({
    id: "wo-low",
    title: "Low score",
    stepIndex: 0,
    appId: "app-implement",
    createdAt: "2026-09-20T00:00:00.000Z",
    executionUpdatedAt: "2026-09-28T00:00:00.000Z",
    confidence: 1,
  });
  const noScore = phaseOrder({
    id: "wo-none",
    title: "No score",
    stepIndex: 0,
    appId: "app-implement",
    createdAt: "2026-09-18T00:00:00.000Z",
    executionUpdatedAt: "2026-09-20T00:00:00.000Z",
  });
  const laterPage = phaseOrder({
    id: "wo-later",
    title: "Later page",
    stepIndex: 0,
    appId: "app-implement",
    createdAt: "2026-06-01T00:00:00.000Z",
    executionUpdatedAt: "2026-09-01T00:00:00.000Z",
    confidence: 5,
    originUrl: "https://github.com/acme/app/issues/9",
  });
  const otherColumn = phaseOrder({
    id: "wo-other",
    title: "Other column",
    stepIndex: 1,
    appId: "app-verify",
    createdAt: "2026-09-28T00:00:00.000Z",
    executionUpdatedAt: "2026-09-28T00:00:00.000Z",
    confidence: 5,
  });
  const orders = [lowScore, noScore, otherColumn, laterPage];

  it("includes a later-page card, keeps a no-score card last, and leaves another column alone", () => {
    const phase = placedPhase(0, orders);
    const other = placedPhase(1, orders);

    expect(phase.map((run) => run.workOrderId)).toEqual(["wo-low", "wo-none", "wo-later"]);
    expect(other.map((run) => run.workOrderId)).toEqual(["wo-other"]);

    const viewed = applyPhaseColumnView(phase, choice({ sortKey: "confidence" }), NOW);

    expect(viewed.map((run) => run.workOrderId)).toEqual(["wo-later", "wo-low", "wo-none"]);
    expect(viewed.map((run) => run.workOrderId)).not.toContain("wo-other");
    expect(other.map((run) => run.workOrderId)).toEqual(["wo-other"]);
  });

  it("hides a lower score and a card with no score when a minimum is set", () => {
    const viewed = applyPhaseColumnView(placedPhase(0, orders), withLineColumnMinimum(choice(), 3), NOW);

    expect(viewed.map((run) => run.workOrderId)).toEqual(["wo-later"]);
  });

  it("shows only cards with no score", () => {
    const viewed = applyPhaseColumnView(placedPhase(0, orders), withLineColumnNoScore(choice(), true), NOW);

    expect(viewed.map((run) => run.workOrderId)).toEqual(["wo-none"]);
  });

  it("keeps no-score cards last when confidence is reversed", () => {
    const viewed = applyPhaseColumnView(
      placedPhase(0, orders),
      choice({ sortKey: "confidence", direction: "reverse" }),
      NOW,
    );

    expect(viewed.map((run) => run.workOrderId)).toEqual(["wo-low", "wo-later", "wo-none"]);
  });

  it("uses work-order created time for age, not execution time", () => {
    const freshExecutionOldIssue = phaseOrder({
      id: "wo-old-issue",
      title: "Old issue",
      stepIndex: 0,
      appId: "app-implement",
      createdAt: "2026-01-01T00:00:00.000Z",
      executionUpdatedAt: "2026-09-28T12:00:00.000Z",
    });
    const staleExecutionNewIssue = phaseOrder({
      id: "wo-new-issue",
      title: "New issue",
      stepIndex: 0,
      appId: "app-implement",
      createdAt: "2026-09-28T00:00:00.000Z",
      executionUpdatedAt: "2026-09-01T00:00:00.000Z",
    });
    const phase = placedPhase(0, [freshExecutionOldIssue, staleExecutionNewIssue]);

    expect(applyPhaseColumnView(phase, withLineColumnSort(choice(), "age"), NOW).map((run) => run.workOrderId)).toEqual(
      ["wo-old-issue", "wo-new-issue"],
    );
    expect(applyPhaseColumnView(phase, withLineColumnAge(choice(), "7"), NOW).map((run) => run.workOrderId)).toEqual([
      "wo-new-issue",
    ]);
    expect(
      applyPhaseColumnView(phase, withLineColumnAge(choice(), "older"), NOW).map((run) => run.workOrderId),
    ).toEqual(["wo-old-issue"]);
  });

  it("does not reorder when the choice is the default", () => {
    const phase = placedPhase(0, orders);

    expect(applyPhaseColumnView(phase, choice(), NOW).map((run) => run.workOrderId)).toEqual(
      phase.map((run) => run.workOrderId),
    );
  });
});

describe("applyOrderColumnView", () => {
  it("keeps placement order by default and reverses work-order update time", () => {
    const older = {
      id: "wo-b",
      title: "Older",
      createdAt: "2026-09-01T00:00:00.000Z",
      updatedAt: "2026-09-10T00:00:00.000Z",
    } as FactoriesWorkOrder;
    const newer = {
      id: "wo-a",
      title: "Newer",
      createdAt: "2026-09-02T00:00:00.000Z",
      updatedAt: "2026-09-20T00:00:00.000Z",
    } as FactoriesWorkOrder;

    expect(applyOrderColumnView([older, newer], choice(), NOW).map((order) => order.id)).toEqual(["wo-b", "wo-a"]);
    expect(
      applyOrderColumnView([newer, older], choice({ sortKey: "updated", direction: "reverse" }), NOW).map(
        (order) => order.id,
      ),
    ).toEqual(["wo-b", "wo-a"]);
  });
});

describe("columnViewReadiness", () => {
  const open: { hasMore: boolean; isLoading: boolean; isError: boolean } = {
    hasMore: false,
    isLoading: false,
    isError: false,
  };

  it("stays idle with no choice and does not wait on later pages", () => {
    expect(columnViewReadiness(choice(), [{ ...open, hasMore: true }])).toBe("idle");
  });

  it("stays pending until the column bucket has no next page", () => {
    const pages = columnViewPageLoads("done", {
      open: { ...open, hasMore: true },
      done: open,
    });

    expect(columnViewReadiness(choice({ sortKey: "age" }), pages)).toBe("pending");
  });

  it("keeps the default order when a later page fails", () => {
    expect(columnViewReadiness(choice({ minimumConfidence: 4 }), [{ ...open, hasMore: true, isError: true }])).toBe(
      "idle",
    );
  });

  it("is ready only after every required page has loaded", () => {
    const pages = columnViewPageLoads("phase", { open, done: { ...open, hasMore: true } });

    expect(columnViewReadiness(choice({ sortKey: "source" }), pages)).toBe("ready");
  });
});
