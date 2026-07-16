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

  it("falls back to the last execution when timestamps are missing", () => {
    expect(
      newestExecution([
        execution({ id: "first", state: "STATE_FINISHED" }),
        execution({ id: "last", state: "STATE_STARTED" }),
      ])?.id,
    ).toBe("last");
  });

  it("uses the last execution when every timestamp is invalid", () => {
    expect(
      newestExecution([
        execution({ id: "first", createdAt: "not-a-date", updatedAt: "also-invalid" }),
        execution({ id: "last", createdAt: "", updatedAt: "" }),
      ])?.id,
    ).toBe("last");
  });
});

describe("hasActiveLiveRuntimeExecutionOnLatest", () => {
  it("detects active state on the latest execution without timestamps", () => {
    expect(
      hasActiveLiveRuntimeExecutionOnLatest([
        execution({ id: "older", state: "STATE_FINISHED" }),
        execution({ id: "latest", state: "STATE_STARTED" }),
      ]),
    ).toBe(true);
  });
});
