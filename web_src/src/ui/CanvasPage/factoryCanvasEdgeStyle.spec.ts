import { Position, type Edge, type Node } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import {
  FACTORY_SPINE_HANDLE_ID,
  factoryRunLeafEdgeKey,
  layoutFactoryRunLeafGraph,
} from "@/lib/layout/factoryRunLeafLayout";
import { buildStyledCanvasEdges } from "./factoryCanvasEdgeStyle";

type RoutedEdge = Edge & {
  sourcePosition?: Position;
  targetPosition?: Position;
};

describe("buildStyledCanvasEdges factory run routing", () => {
  it("uses compact ports, edge badges, and distinct gutters for feedback channels", () => {
    const nodes: Node[] = [
      { id: "loop", position: { x: 0, y: 0 }, data: {} },
      { id: "workflow", position: { x: 0, y: 300 }, data: {} },
      { id: "runner", position: { x: 400, y: 500 }, data: {} },
    ];
    const edges: Edge[] = [
      { id: "loop-workflow", source: "loop", target: "workflow", sourceHandle: "next" },
      { id: "workflow-loop", source: "workflow", target: "loop", sourceHandle: "passed" },
      { id: "workflow-runner", source: "workflow", target: "runner", sourceHandle: "failed" },
      { id: "runner-loop-passed", source: "runner", target: "loop", sourceHandle: "passed" },
      { id: "runner-loop-failed", source: "runner", target: "loop", sourceHandle: "failed" },
    ];
    const layout = layoutFactoryRunLeafGraph(
      nodes.map((node) => ({ id: node.id, position: node.position })),
      edges,
    );

    const styledEdges = buildStyledCanvasEdges({
      edges,
      nodes,
      isVerticalFlow: true,
      resolvedThemeIsDark: false,
      edgeDefaults: { type: "custom", style: { stroke: "#cbd5e1", strokeWidth: 1.5 } },
      hoveredEdgeId: null,
      isEditMode: false,
      isReadOnly: false,
      stableEdgeDelete: vi.fn(),
      factoryRunLeafLayout: layout,
    })!;

    const passed = styledEdges.find((edge) => edge.id === "runner-loop-passed")! as RoutedEdge;
    const failed = styledEdges.find((edge) => edge.id === "runner-loop-failed")! as RoutedEdge;
    const passedGutter = layout.edgeRouteGutters.get(factoryRunLeafEdgeKey("runner", "loop", "passed"));
    const failedGutter = layout.edgeRouteGutters.get(factoryRunLeafEdgeKey("runner", "loop", "failed"));
    const passedOffsetY = layout.edgeRouteOffsetsY.get(factoryRunLeafEdgeKey("runner", "loop", "passed"));
    const failedOffsetY = layout.edgeRouteOffsetsY.get(factoryRunLeafEdgeKey("runner", "loop", "failed"));

    expect(passed.sourceHandle).toBe(FACTORY_SPINE_HANDLE_ID);
    expect(failed.sourceHandle).toBe(FACTORY_SPINE_HANDLE_ID);
    expect(passed.sourcePosition).toBe(Position.Bottom);
    expect(passed.targetPosition).toBe(Position.Top);
    expect(passed.data).toMatchObject({
      channelLabel: "passed",
      routeGutterX: passedGutter,
      routeOffsetY: passedOffsetY,
    });
    expect(failed.data).toMatchObject({
      channelLabel: "failed",
      routeGutterX: failedGutter,
      routeOffsetY: failedOffsetY,
    });
    expect(passedGutter).not.toBe(failedGutter);
    expect(passedOffsetY).not.toBe(failedOffsetY);
  });
});
