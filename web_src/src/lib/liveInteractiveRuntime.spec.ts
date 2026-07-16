import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { hasActiveLiveRuntimeExecutionOnLatest, newestExecution } from "./liveInteractiveRuntime";

function execution(overrides: Partial<CanvasesCanvasNodeExecution>): CanvasesCanvasNodeExecution {
  return overrides as CanvasesCanvasNodeExecution;
}

describe("newestExecution", () => {
  it("prefers executions with valid timestamps", () => {
    expect(
      newestExecution([
        execution({ id: "older", createdAt: "2026-07-07T10:00:00Z" }),
        execution({ id: "latest", createdAt: "2026-07-07T10:05:00Z" }),
      ])?.id,
    ).toBe("latest");
  });

  it("prefers createdAt over a newer updatedAt on an older execution", () => {
    expect(
      newestExecution([
        execution({
          id: "active",
          createdAt: "2026-07-07T10:20:00Z",
          updatedAt: "2026-07-07T10:20:00Z",
          state: "STATE_STARTED",
        }),
        execution({
          id: "finished",
          createdAt: "2026-07-07T10:10:00Z",
          updatedAt: "2026-07-07T10:25:00Z",
          state: "STATE_FINISHED",
        }),
      ])?.id,
    ).toBe("active");
  });

  it("falls back to the first execution when timestamps are missing", () => {
    expect(
      newestExecution([
        execution({ id: "newest", state: "STATE_STARTED" }),
        execution({ id: "older", state: "STATE_FINISHED" }),
      ])?.id,
    ).toBe("newest");
  });

  it("uses the first execution when every timestamp is invalid", () => {
    expect(
      newestExecution([
        execution({ id: "newest", createdAt: "not-a-date", updatedAt: "also-invalid" }),
        execution({ id: "older", createdAt: "", updatedAt: "" }),
      ])?.id,
    ).toBe("newest");
  });
});

describe("hasActiveLiveRuntimeExecutionOnLatest", () => {
  it("detects active state on the latest execution without timestamps", () => {
    expect(
      hasActiveLiveRuntimeExecutionOnLatest([
        execution({ id: "latest", state: "STATE_STARTED" }),
        execution({ id: "older", state: "STATE_FINISHED" }),
      ]),
    ).toBe(true);
  });
});
