import { describe, expect, it } from "vitest";
import {
  buildAgentStagingAutoOpenKey,
  claimAgentStagingAutoOpen,
  releaseAgentStagingAutoOpen,
} from "./agent-staging-auto-open";

describe("agent-staging-auto-open", () => {
  it("builds a stable key from canvas id and message", () => {
    expect(buildAgentStagingAutoOpenKey("canvas-1", "ready")).toBe("canvas-1:ready");
    expect(buildAgentStagingAutoOpenKey("canvas-1")).toBe("canvas-1:");
  });

  it("claims a staging key only once until released", () => {
    const key = "canvas-1:ready";

    expect(claimAgentStagingAutoOpen(key)).toBe(true);
    expect(claimAgentStagingAutoOpen(key)).toBe(false);

    releaseAgentStagingAutoOpen(key);

    expect(claimAgentStagingAutoOpen(key)).toBe(true);
  });
});
