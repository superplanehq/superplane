import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { useMentionCandidates } from "./useMentionCandidates";

const nodes = [
  { id: "node-1", name: "Deploy API", component: "http", type: "TYPE_COMPONENT" },
  { id: "node-2", name: "Start", component: "webhook", type: "TYPE_TRIGGER" },
] as SuperplaneComponentsNode[];

const runs = [{ id: "abcdef123", result: "RESULT_PASSED", createdAt: new Date().toISOString() }] as CanvasesCanvasRun[];

describe("useMentionCandidates", () => {
  it("returns a stable empty list while candidate lookup is disabled", () => {
    const { result, rerender } = renderHook(({ filter }) => useMentionCandidates(nodes, runs, filter, false), {
      initialProps: { filter: "" },
    });
    const initialCandidates = result.current;

    rerender({ filter: "deploy" });

    expect(result.current).toBe(initialCandidates);
    expect(result.current).toEqual([]);
  });

  it("builds filtered candidates when lookup is enabled", () => {
    const { result } = renderHook(() => useMentionCandidates(nodes, runs, "deploy", true));

    expect(result.current).toEqual([
      expect.objectContaining({
        type: "node",
        id: "node-1",
        label: "Deploy API",
      }),
    ]);
  });
});
