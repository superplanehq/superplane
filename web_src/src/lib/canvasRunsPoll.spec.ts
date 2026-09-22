import { describe, expect, it } from "bun:test";
import { CANVAS_RUNS_POLL_MS, canvasRunsPollInterval } from "./canvasRunsPoll";

describe("canvasRunsPollInterval", () => {
  it("stops ListRuns polling while the canvas websocket is connected", () => {
    expect(canvasRunsPollInterval(true)).toBe(false);
  });

  it("polls ListRuns when the canvas websocket is down", () => {
    expect(canvasRunsPollInterval(false)).toBe(CANVAS_RUNS_POLL_MS);
  });
});
