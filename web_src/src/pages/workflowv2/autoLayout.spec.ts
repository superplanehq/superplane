import { describe, expect, it } from "vitest";
import type { CanvasesCanvas } from "@/api-client";
import { applyHorizontalAutoLayout } from "./autoLayout";

describe("applyHorizontalAutoLayout", () => {
  it("does not crash when an edge channel is missing from channelsByNodeId", async () => {
    const workflow: CanvasesCanvas = {
      metadata: {
        id: "canvas-1",
        name: "regression-canvas",
      },
      spec: {
        nodes: [
          {
            id: "list-main-changes",
            name: "List Main Changes",
            type: "TYPE_COMPONENT",
            component: { name: "github.list_main_changes" },
            position: { x: 40, y: 80 },
          },
          {
            id: "notify",
            name: "Notify",
            type: "TYPE_COMPONENT",
            component: { name: "slack.send_text_message" },
            position: { x: 560, y: 80 },
          },
        ],
        edges: [
          {
            sourceId: "list-main-changes",
            targetId: "notify",
            channel: "success",
          },
        ],
      },
    };

    const channelsByNodeId = new Map<string, string[]>([
      ["list-main-changes", ["default"]],
      ["notify", ["default"]],
    ]);

    await expect(
      applyHorizontalAutoLayout(workflow, {
        scope: "connected-component",
        nodeIds: ["list-main-changes"],
        channelsByNodeId,
      }),
    ).resolves.toMatchObject({
      spec: {
        edges: [
          {
            sourceId: "list-main-changes",
            targetId: "notify",
            channel: "success",
          },
        ],
      },
    });
  });
});
