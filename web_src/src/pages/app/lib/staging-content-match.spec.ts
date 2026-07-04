import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { matchesCommittedCanvasYaml, matchesCommittedConsoleYaml } from "./staging-content-match";

const emptyConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels: []\n  layout: []\n";

const sampleCanvasYaml = `apiVersion: v1
kind: Canvas
metadata:
  name: demo
spec:
  nodes:
    - id: node-1
      name: Trigger
      type: trigger
  edges: []
`;

const reorderedCanvasYaml = `apiVersion: v1
kind: Canvas
metadata:
  name: demo
spec:
  edges: []
  nodes:
    - id: node-1
      name: Trigger
      type: trigger
`;

describe("staging-content-match", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        if (url.includes("canvas.yaml") && !url.includes("stage=true")) {
          return new Response(sampleCanvasYaml, { status: 200 });
        }
        if (url.includes("console.yaml") && !url.includes("stage=true")) {
          return new Response(emptyConsoleYaml, { status: 200 });
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("treats semantically identical canvas yaml as committed", async () => {
    await expect(matchesCommittedCanvasYaml("canvas-1", "version-1", reorderedCanvasYaml)).resolves.toBe(true);
  });

  it("treats semantically identical console yaml as committed", async () => {
    await expect(matchesCommittedConsoleYaml("canvas-1", "version-1", emptyConsoleYaml)).resolves.toBe(true);
  });
});
