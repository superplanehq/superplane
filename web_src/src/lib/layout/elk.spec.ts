import { describe, expect, it } from "vitest";
import type { CanvasesCanvas, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { ElkLayoutEngine } from "@/lib/layout";
import { resolveForwardLayoutEdges } from "./layoutGraph";

type ElkGraphForTest = {
  children?: Array<{
    id: string;
    ports?: Array<{ id: string }>;
  }>;
};

type ElkLayoutEngineInternals = {
  buildElkGraph(
    workflow: CanvasesCanvas,
    layoutNodes: ComponentsNode[],
    outputChannelsByNodeId: Map<string, string[]>,
  ): ElkGraphForTest;
};

describe("ElkLayoutEngine", () => {
  it("does not crash when an edge channel is missing from outputChannelsByNodeId", async () => {
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
            type: "TYPE_ACTION",
            component: "github.list_main_changes",
            position: { x: 40, y: 80 },
          },
          {
            id: "notify",
            name: "Notify",
            type: "TYPE_ACTION",
            component: "slack.send_text_message",
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

    const autoLayout = new ElkLayoutEngine();
    await expect(
      autoLayout.apply(workflow, {
        scope: "connected-component",
        nodeIds: ["list-main-changes"],
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
            type: "TYPE_ACTION",
            component: "comp.a1",
            position: { x: 0, y: 0 },
          },
          {
            id: "component-a-2",
            name: "A2",
            type: "TYPE_ACTION",
            component: "comp.a2",
            position: { x: 300, y: 0 },
          },
          {
            id: "component-b-1",
            name: "B1",
            type: "TYPE_ACTION",
            component: "comp.b1",
            position: { x: 0, y: 500 },
          },
          {
            id: "component-b-2",
            name: "B2",
            type: "TYPE_ACTION",
            component: "comp.b2",
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

    const autoLayout = new ElkLayoutEngine();
    const result = await autoLayout.apply(workflow, {
      scope: "full-canvas",
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

  it("preserves forward flow when a component has a loop-back edge", async () => {
    const workflow: CanvasesCanvas = {
      metadata: {
        id: "canvas-3",
        name: "loop-layout",
      },
      spec: {
        nodes: [
          {
            id: "start",
            name: "Start",
            type: "TYPE_ACTION",
            component: "comp.start",
            position: { x: 0, y: 0 },
          },
          {
            id: "process",
            name: "Process",
            type: "TYPE_ACTION",
            component: "comp.process",
            position: { x: 600, y: 0 },
          },
          {
            id: "check",
            name: "Check",
            type: "TYPE_ACTION",
            component: "comp.check",
            position: { x: 1200, y: 0 },
          },
        ],
        edges: [
          { sourceId: "start", targetId: "process", channel: "default" },
          { sourceId: "process", targetId: "check", channel: "default" },
          { sourceId: "check", targetId: "start", channel: "repeat" },
        ],
      },
    };

    const autoLayout = new ElkLayoutEngine();
    const result = await autoLayout.apply(workflow, { scope: "full-canvas" });
    const byId = new Map((result.spec?.nodes || []).map((node) => [node.id!, node]));

    expect(byId.get("start")!.position!.x!).toBeLessThan(byId.get("process")!.position!.x!);
    expect(byId.get("process")!.position!.x!).toBeLessThan(byId.get("check")!.position!.x!);
  });

  it("preserves component output channel order before edge-discovered channels", () => {
    const workflow: CanvasesCanvas = {
      metadata: {
        id: "canvas-4",
        name: "channel-order",
      },
      spec: {
        nodes: [
          {
            id: "source",
            name: "Source",
            type: "TYPE_ACTION",
            component: "runner",
            position: { x: 0, y: 0 },
          },
          {
            id: "target",
            name: "Target",
            type: "TYPE_ACTION",
            component: "noop",
            position: { x: 600, y: 0 },
          },
        ],
        edges: [
          { sourceId: "source", targetId: "target", channel: "failed" },
          { sourceId: "source", targetId: "target", channel: "passed" },
        ],
      },
    };

    const autoLayout = new ElkLayoutEngine();
    const graph = (autoLayout as unknown as ElkLayoutEngineInternals).buildElkGraph(
      workflow,
      workflow.spec!.nodes!,
      new Map([["source", ["passed", "failed"]]]),
    );
    const source = graph.children?.find((child) => child.id === "source");

    expect(source?.ports?.map((port) => port.id)).toEqual(["source__input", "source__passed", "source__failed"]);
  });

  it("keeps layout edges when node positions are missing", () => {
    const edges = [{ sourceId: "source", targetId: "target", channel: "default" }];

    expect(resolveForwardLayoutEdges([{ id: "source" }, { id: "target" }], edges)).toEqual(edges);
  });

  it("preserves a forward edge into a new node at the origin when a loop exists", () => {
    const edges = [
      { sourceId: "process", targetId: "check", channel: "default" },
      { sourceId: "check", targetId: "new-node", channel: "default" },
      { sourceId: "new-node", targetId: "process", channel: "repeat" },
    ];

    expect(
      resolveForwardLayoutEdges(
        [
          { id: "process", position: { x: 600, y: 0 } },
          { id: "check", position: { x: 1200, y: 0 } },
          { id: "new-node", position: { x: 0, y: 0 } },
        ],
        edges,
      ),
    ).toEqual(edges.slice(0, 2));
  });
});
