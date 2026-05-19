import { describe, expect, it, vi } from "vitest";
import { makeComponentsNode } from "@/test/factories";
import { mapCanvasNodesToLogEntries } from "./utils";

describe("mapCanvasNodesToLogEntries", () => {
  it("maps node warnings into canvas log entries", () => {
    const entries = mapCanvasNodesToLogEntries({
      nodes: [
        makeComponentsNode({
          id: "draft-node-newer",
          name: "Draft Node Newer",
          warningMessage: "Newer warning",
        }),
        makeComponentsNode({
          id: "draft-node-older",
          name: "Draft Node Older",
          warningMessage: "Older warning",
        }),
      ],
      workflowUpdatedAt: "2026-04-03T12:00:00Z",
      onNodeSelect: vi.fn(),
    });

    expect(entries).toHaveLength(2);
    expect(entries.map((entry) => entry.id)).toEqual(["warning-1", "warning-2"]);
    expect(entries.every((entry) => entry.type === "warning")).toBe(true);
    expect(entries.every((entry) => entry.source === "canvas")).toBe(true);
    expect(entries[1]?.searchText).toContain("Older warning");
  });
});
