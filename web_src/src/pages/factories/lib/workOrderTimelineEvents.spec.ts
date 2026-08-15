import { describe, expect, it } from "vitest";
import type { FactoriesWorkOrderExecution } from "@/api-client";
import { formatStepExecutionDuration, type WorkOrderTimelineStep } from "./workOrderTimelineEvents";

function buildStep(overrides: Partial<WorkOrderTimelineStep> = {}): WorkOrderTimelineStep {
  return {
    id: "step-1",
    stepName: "Build",
    at: "2026-08-12T10:00:00.000Z",
    startedAt: "2026-08-12T10:00:00.000Z",
    finishedAt: "2026-08-12T10:00:16.482Z",
    execution: {} as FactoriesWorkOrderExecution,
    ...overrides,
  };
}

describe("formatStepExecutionDuration", () => {
  it("returns null when startedAt is missing", () => {
    const step = buildStep({ startedAt: "" });

    expect(formatStepExecutionDuration(step)).toBeNull();
  });

  it("returns null when finishedAt is missing", () => {
    const step = buildStep({ finishedAt: undefined });

    expect(formatStepExecutionDuration(step)).toBeNull();
  });

  it("returns null for a zero or negative duration", () => {
    const step = buildStep({
      startedAt: "2026-08-12T10:00:00.000Z",
      finishedAt: "2026-08-12T10:00:00.000Z",
    });

    expect(formatStepExecutionDuration(step)).toBeNull();
  });

  it("renders '< 1s' for sub-second durations instead of milliseconds", () => {
    const step = buildStep({
      startedAt: "2026-08-12T10:00:00.000Z",
      finishedAt: "2026-08-12T10:00:00.482Z",
    });

    expect(formatStepExecutionDuration(step)).toBe("< 1s");
  });

  it("renders whole seconds for multi-second durations without a milliseconds suffix", () => {
    const step = buildStep({
      startedAt: "2026-08-12T10:00:00.000Z",
      finishedAt: "2026-08-12T10:00:16.482Z",
    });

    expect(formatStepExecutionDuration(step)).toBe("16s");
  });
});
