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

  it("stacks disconnected components vertically", async () => {
    const workflow: CanvasesCanvas = {
      metadata: {
        id: "canvas-2",
        name: "disconnected-layout",
      },
      spec: {
        nodes: [
          {
            id: "component-a-1",
            name: "A1",
            type: "TYPE_COMPONENT",
            component: { name: "comp.a1" },
            position: { x: 0, y: 0 },
          },
          {
            id: "component-a-2",
            name: "A2",
            type: "TYPE_COMPONENT",
            component: { name: "comp.a2" },
            position: { x: 300, y: 0 },
          },
          {
            id: "component-b-1",
            name: "B1",
            type: "TYPE_COMPONENT",
            component: { name: "comp.b1" },
            position: { x: 0, y: 500 },
          },
          {
            id: "component-b-2",
            name: "B2",
            type: "TYPE_COMPONENT",
            component: { name: "comp.b2" },
            position: { x: 300, y: 500 },
          },
        ],
        edges: [
          {
            sourceId: "component-a-1",
            targetId: "component-a-2",
            channel: "default",
          },
          {
            sourceId: "component-b-1",
            targetId: "component-b-2",
            channel: "default",
          },
        ],
      },
    };

    const channelsByNodeId = new Map<string, string[]>([
      ["component-a-1", ["default"]],
      ["component-a-2", ["default"]],
      ["component-b-1", ["default"]],
      ["component-b-2", ["default"]],
    ]);

    const result = await applyHorizontalAutoLayout(workflow, {
      scope: "full-canvas",
      channelsByNodeId,
    });

    const byId = new Map((result.spec?.nodes || []).map((node) => [node.id!, node]));
    const a1 = byId.get("component-a-1");
    const a2 = byId.get("component-a-2");
    const b1 = byId.get("component-b-1");
    const b2 = byId.get("component-b-2");

    expect(a1?.position).toBeDefined();
    expect(a2?.position).toBeDefined();
    expect(b1?.position).toBeDefined();
    expect(b2?.position).toBeDefined();

    const componentAMaxY = Math.max(a1!.position!.y! + 180, a2!.position!.y! + 180);
    const componentBMinY = Math.min(b1!.position!.y!, b2!.position!.y!);

    expect(componentBMinY).toBeGreaterThan(componentAMaxY);

    const componentAMinX = Math.min(a1!.position!.x!, a2!.position!.x!);
    const componentBMinX = Math.min(b1!.position!.x!, b2!.position!.x!);

    expect(Math.abs(componentAMinX - componentBMinX)).toBeLessThanOrEqual(1);
  });
});
