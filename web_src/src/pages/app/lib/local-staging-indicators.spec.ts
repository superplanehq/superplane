import { describe, expect, it } from "vitest";

import type { CanvasesCanvas } from "@/api-client";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import type { PendingFileChange } from "../files/types";
import { hasLocalCanvasGraphDiff, hasLocalConsoleDiff, hasLocalFilesStaging } from "./local-staging-indicators";

describe("local-staging-indicators", () => {
  it("detects canvas graph diffs using semantic comparison", () => {
    const committed = {
      nodes: [{ id: "n1", name: "A", type: "trigger" }],
      edges: [],
    } as unknown as CanvasesCanvas["spec"];
    const effectiveSame = {
      edges: [],
      nodes: [{ id: "n1", name: "A", type: "trigger" }],
    } as unknown as CanvasesCanvas["spec"];
    const effectiveDifferent = {
      nodes: [{ id: "n1", name: "B", type: "trigger" }],
      edges: [],
    } as unknown as CanvasesCanvas["spec"];

    expect(hasLocalCanvasGraphDiff(committed, effectiveSame)).toBe(false);
    expect(hasLocalCanvasGraphDiff(committed, effectiveDifferent)).toBe(true);
  });

  it("detects console diffs from effective local state", () => {
    const committed = {
      panels: [{ id: "p1", type: "markdown", content: { body: "hi" } }],
      layout: [],
    } as { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] };
    const effectiveSame = {
      panels: [{ id: "p1", type: "markdown", content: { body: "hi" } }],
      layout: [],
    } as { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] };
    const effectiveDifferent = {
      panels: [{ id: "p1", type: "markdown", content: { body: "bye" } }],
      layout: [],
    } as { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] };

    expect(hasLocalConsoleDiff(committed, effectiveSame)).toBe(false);
    expect(hasLocalConsoleDiff(committed, effectiveDifferent)).toBe(true);
  });

  it("detects files staging from pending changes vs committed baseline", () => {
    const pending: PendingFileChange[] = [{ type: "modified", path: "README.md", content: "draft" }];
    const committed = { "README.md": "committed" };

    expect(hasLocalFilesStaging(pending, committed)).toBe(true);
    expect(hasLocalFilesStaging([], committed)).toBe(false);
    expect(hasLocalFilesStaging(pending, { "README.md": "draft" })).toBe(false);
  });
});
