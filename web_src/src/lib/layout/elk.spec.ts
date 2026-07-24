import { describe, expect, it } from "vitest";
import type { CanvasesCanvas, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { ElkLayoutEngine } from "@/lib/layout";
import type { LayoutDirection } from "@/lib/layout";
import { resolveForwardLayoutEdges } from "./layoutGraph";

type ElkGraphForTest = {
  layoutOptions?: Record<string, string>;
  children?: Array<{
    id: string;
    ports?: Array<{ id: string; properties?: Record<string, string> }>;
  }>;
};

type ElkLayoutEngineInternals = {
  buildElkGraph(
    workflow: CanvasesCanvas,
    layoutNodes: ComponentsNode[],
    outputChannelsByNodeId: Map<string, string[]>,
    direction: LayoutDirection,
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
      "RIGHT",
    );
    const source = graph.children?.find((child) => child.id === "source");

    expect(source?.ports?.map((port) => port.id)).toEqual(["source__input", "source__passed", "source__failed"]);
  });

  it("uses top/bottom ports and DOWN direction for vertical layout", () => {
    const workflow: CanvasesCanvas = {
      metadata: { id: "canvas-vertical", name: "vertical-ports" },
      spec: {
        nodes: [
          { id: "source", name: "Source", type: "TYPE_ACTION", component: "runner", position: { x: 0, y: 0 } },
          { id: "target", name: "Target", type: "TYPE_ACTION", component: "noop", position: { x: 0, y: 600 } },
        ],
        edges: [{ sourceId: "source", targetId: "target", channel: "default" }],
      },
    };

    const autoLayout = new ElkLayoutEngine();
    const graph = (autoLayout as unknown as ElkLayoutEngineInternals).buildElkGraph(
      workflow,
      workflow.spec!.nodes!,
      new Map([["source", ["default"]]]),
      "DOWN",
    );

    expect(graph.layoutOptions?.["elk.direction"]).toBe("DOWN");
    const source = graph.children?.find((child) => child.id === "source");
    const inputPort = source?.ports?.find((port) => port.id === "source__input");
    const outputPort = source?.ports?.find((port) => port.id === "source__default");
    expect(inputPort?.properties?.["elk.port.side"]).toBe("NORTH");
    expect(outputPort?.properties?.["elk.port.side"]).toBe("SOUTH");
  });

  it("lays connected nodes top-to-bottom in DOWN direction", async () => {
    const workflow: CanvasesCanvas = {
      metadata: { id: "canvas-vertical-flow", name: "vertical-flow" },
      spec: {
        nodes: [
          { id: "start", name: "Start", type: "TYPE_ACTION", component: "comp.start", position: { x: 0, y: 0 } },
          { id: "middle", name: "Middle", type: "TYPE_ACTION", component: "comp.middle", position: { x: 0, y: 0 } },
          { id: "end", name: "End", type: "TYPE_ACTION", component: "comp.end", position: { x: 0, y: 0 } },
        ],
        edges: [
          { sourceId: "start", targetId: "middle", channel: "default" },
          { sourceId: "middle", targetId: "end", channel: "default" },
        ],
      },
    };

    const autoLayout = new ElkLayoutEngine();
    const positions = await autoLayout.computeLayoutPositions(workflow, { direction: "DOWN" });

    expect(positions.size).toBe(3);
    expect(positions.get("start")!.y).toBeLessThan(positions.get("middle")!.y);
    expect(positions.get("middle")!.y).toBeLessThan(positions.get("end")!.y);
    // Nodes stay in a single vertical column.
    expect(Math.abs(positions.get("start")!.x - positions.get("end")!.x)).toBeLessThanOrEqual(1);
  });

  it("stacks disconnected components side-by-side in DOWN direction", async () => {
    const workflow: CanvasesCanvas = {
      metadata: { id: "canvas-vertical-disconnected", name: "vertical-disconnected" },
      spec: {
        nodes: [
          { id: "a1", name: "A1", type: "TYPE_ACTION", component: "comp.a1", position: { x: 0, y: 0 } },
          { id: "a2", name: "A2", type: "TYPE_ACTION", component: "comp.a2", position: { x: 0, y: 300 } },
          { id: "b1", name: "B1", type: "TYPE_ACTION", component: "comp.b1", position: { x: 500, y: 0 } },
          { id: "b2", name: "B2", type: "TYPE_ACTION", component: "comp.b2", position: { x: 500, y: 300 } },
        ],
        edges: [
          { sourceId: "a1", targetId: "a2", channel: "default" },
          { sourceId: "b1", targetId: "b2", channel: "default" },
        ],
      },
    };

    const autoLayout = new ElkLayoutEngine();
    const positions = await autoLayout.computeLayoutPositions(workflow, {
      direction: "DOWN",
      scope: "full-canvas",
    });

    const componentAMaxX = Math.max(positions.get("a1")!.x + 420, positions.get("a2")!.x + 420);
    const componentBMinX = Math.min(positions.get("b1")!.x, positions.get("b2")!.x);
    expect(componentBMinX).toBeGreaterThan(componentAMaxX);
  });

  it("does not mutate the workflow when computing overlay positions", async () => {
    const workflow: CanvasesCanvas = {
      metadata: { id: "canvas-overlay", name: "overlay" },
      spec: {
        nodes: [
          { id: "start", name: "Start", type: "TYPE_ACTION", component: "comp.start", position: { x: 10, y: 20 } },
          { id: "end", name: "End", type: "TYPE_ACTION", component: "comp.end", position: { x: 30, y: 40 } },
        ],
        edges: [{ sourceId: "start", targetId: "end", channel: "default" }],
      },
    };

    const autoLayout = new ElkLayoutEngine();
    await autoLayout.computeLayoutPositions(workflow, { direction: "DOWN" });

    expect(workflow.spec?.nodes?.[0].position).toEqual({ x: 10, y: 20 });
    expect(workflow.spec?.nodes?.[1].position).toEqual({ x: 30, y: 40 });
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
