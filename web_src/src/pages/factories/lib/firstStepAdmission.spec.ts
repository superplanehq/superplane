import { describe, expect, it } from "bun:test";
import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import {
  countInFlightExecutionsAtStep,
  reservedStartCount,
  willQueueFirstStepStart,
  type FirstStepAdmission,
} from "./firstStepAdmission";

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "poc",
  steps: [{ app: { app: "app-plan" }, maxParallelism: 1 }, { app: { app: "app-build" } }],
};

const ROOMY: FactoriesFactoryLine = {
  ...LINE,
  steps: [{ app: { app: "app-plan" }, maxParallelism: 10 }, { app: { app: "app-build" } }],
};

function order(id: string, dispatch: NonNullable<FactoriesWorkOrder["lineDispatches"]>[number]): FactoriesWorkOrder {
  return {
    id,
    title: id,
    state: "STATE_OPEN",
    lineDispatches: [dispatch],
  };
}

function runningAtFirstStep(id: string): FactoriesWorkOrder {
  return order(id, {
    id: `dispatch-${id}`,
    line: { id: LINE.id, name: LINE.name },
    state: "STATE_ACTIVE",
    stepExecutions: [
      {
        id: `exec-${id}`,
        step: "plan",
        stepIndex: 0,
        state: "STATE_STARTED",
      },
    ],
  });
}

function willQueue(workOrders: FactoriesWorkOrder[], extra?: Partial<FirstStepAdmission>): boolean {
  return willQueueFirstStepStart({ line: LINE, workOrders, ...extra });
}

describe("willQueueFirstStepStart", () => {
  it("returns false on an empty line", () => {
    expect(willQueue([])).toBe(false);
  });

  it("returns true when in-flight work fills the first-step cap", () => {
    expect(willQueue([runningAtFirstStep("wo-running")])).toBe(true);
  });

  it("returns false when a finished first-step run leaves a free slot", () => {
    const finished = order("wo-done", {
      id: "dispatch-wo-done",
      line: { id: LINE.id, name: LINE.name },
      state: "STATE_ACTIVE",
      stepExecutions: [
        {
          id: "exec-wo-done",
          step: "plan",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
        },
      ],
    });
    expect(willQueue([finished])).toBe(false);
  });

  it("returns true when a waiter already sits at the first step", () => {
    const queued = order("wo-queued", {
      id: "dispatch-wo-queued",
      line: { id: LINE.id, name: LINE.name },
      state: "STATE_ACTIVE",
      stepExecutions: [],
      queueItem: { id: "q-1", stepName: "plan", stepIndex: 0, position: 1 },
    });
    expect(willQueue([queued])).toBe(true);
  });

  it("returns true for FIFO even when a first-step slot is free", () => {
    const queued = order("wo-queued", {
      id: "dispatch-wo-queued",
      line: { id: LINE.id, name: LINE.name },
      state: "STATE_ACTIVE",
      stepExecutions: [],
      queueItem: { id: "q-1", stepName: "plan", stepIndex: 0, position: 1 },
    });
    expect(willQueue([queued], { line: ROOMY })).toBe(true);
  });

  it("does not treat a later-step waiter as a first-step queue", () => {
    const queuedLater = order("wo-queued-build", {
      id: "dispatch-wo-queued-build",
      line: { id: LINE.id, name: LINE.name },
      state: "STATE_ACTIVE",
      stepExecutions: [
        {
          id: "exec-plan",
          step: "plan",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
        },
      ],
      queueItem: { id: "q-2", stepName: "build", stepIndex: 1, position: 1 },
    });
    expect(willQueue([queuedLater])).toBe(false);
  });

  it("does not count queued-only rows as in-flight slot use", () => {
    const queued = order("wo-queued", {
      id: "dispatch-wo-queued",
      line: { id: LINE.id, name: LINE.name },
      state: "STATE_ACTIVE",
      stepExecutions: [],
      queueItem: { id: "q-1", stepName: "plan", stepIndex: 0, position: 1 },
    });
    expect(countInFlightExecutionsAtStep([queued], LINE.id ?? "", 0)).toBe(0);
  });

  it("uses an explicit first-step cap when the column editor overrides it", () => {
    expect(willQueue([runningAtFirstStep("wo-running")], { line: ROOMY, firstStepMaxParallelism: 1 })).toBe(true);
    expect(willQueue([runningAtFirstStep("wo-running")], { firstStepMaxParallelism: 10 })).toBe(false);
  });

  it("returns true when factory-wide in-flight work fills the factory cap", () => {
    expect(
      willQueue([runningAtFirstStep("wo-running")], { line: ROOMY, factoryMaxParallelTasks: 1 }),
    ).toBe(true);
  });

  it("returns true when a Start in flight fills the last factory slot", () => {
    expect(willQueue([], { line: ROOMY, factoryMaxParallelTasks: 1, reservedStarts: 1 })).toBe(true);
  });
});

describe("reservedStartCount", () => {
  it("counts a busy Start that has not created an execution yet", () => {
    expect(reservedStartCount([{ id: "wo-draft", state: "STATE_DRAFT" }], new Set(["wo-draft"]))).toBe(1);
  });

  it("does not double-count a busy Start that already occupies a slot", () => {
    expect(reservedStartCount([runningAtFirstStep("wo-running")], new Set(["wo-running"]))).toBe(0);
  });
});
