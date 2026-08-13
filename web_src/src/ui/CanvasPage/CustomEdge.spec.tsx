import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { TOUCHING_EDGE_STROKE } from "./edgePath";

const { setEdges } = vi.hoisted(() => ({
  setEdges: vi.fn(),
}));

vi.mock("@xyflow/react", () => ({
  BaseEdge: ({ className }: { className?: string }) => <div data-testid="base-edge" data-class-name={className} />,
  EdgeLabelRenderer: ({ children }: { children?: ReactNode }) => <>{children}</>,
  getBezierPath: () => ["M0,0 C10,0 20,10 30,10", 15, 5],
  getSmoothStepPath: () => ["M0,0 L10,0 L10,10 L30,10", 15, 5],
  Position: {
    Left: "left",
    Right: "right",
    Top: "top",
    Bottom: "bottom",
  },
  useReactFlow: () => ({
    setEdges,
  }),
}));

import { CustomEdge } from "./CustomEdge";

function renderEdge(ui: ReactNode) {
  return render(<svg>{ui}</svg>);
}

describe("CustomEdge", () => {
  it("does not show the delete icon or delete on pointer down in live mode", () => {
    const onDelete = vi.fn();

    renderEdge(
      <CustomEdge
        id="edge-1"
        source="node-a"
        target="node-b"
        sourceX={0}
        sourceY={0}
        targetX={10}
        targetY={10}
        sourcePosition={"right" as never}
        targetPosition={"left" as never}
        selected={false}
        data={{ isHovered: true, canDelete: false, onDelete }}
      />,
    );

    expect(screen.queryByTestId("edge-delete-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("edge-delete-hit-area")).not.toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("shows the delete icon and deletes the edge in edit mode", () => {
    const onDelete = vi.fn();

    renderEdge(
      <CustomEdge
        id="edge-1"
        source="node-a"
        target="node-b"
        sourceX={0}
        sourceY={0}
        targetX={10}
        targetY={10}
        sourcePosition={"right" as never}
        targetPosition={"left" as never}
        selected={false}
        data={{ isHovered: true, canDelete: true, onDelete }}
      />,
    );

    expect(screen.getByTestId("edge-delete-icon")).toBeInTheDocument();
    const hitArea = screen.getByTestId("edge-delete-hit-area");

    fireEvent.pointerDown(hitArea, { button: 0, buttons: 1 });
    expect(onDelete).toHaveBeenCalledWith("edge-1");
  });

  it("moves channel labels near the source only when the edge touches another", () => {
    const { rerender } = renderEdge(
      <CustomEdge
        id="edge-touch"
        source="node-a"
        target="node-b"
        sourceX={100}
        sourceY={100}
        targetX={400}
        targetY={300}
        sourcePosition={"right" as never}
        targetPosition={"left" as never}
        selected={false}
        data={{ channelLabel: "false", touchesOtherEdge: true, contrastStroke: true }}
      />,
    );

    const touchingLabel = screen.getByTestId("edge-channel-label");
    expect(touchingLabel).toHaveTextContent("false");
    expect(touchingLabel).toHaveStyle({ borderColor: TOUCHING_EDGE_STROKE });
    const touchingTransform = touchingLabel.style.transform;

    rerender(
      <svg>
        <CustomEdge
          id="edge-touch"
          source="node-a"
          target="node-b"
          sourceX={100}
          sourceY={100}
          targetX={400}
          targetY={300}
          sourcePosition={"right" as never}
          targetPosition={"left" as never}
          selected={false}
          data={{ channelLabel: "false" }}
        />
      </svg>,
    );

    const midLabel = screen.getByTestId("edge-channel-label");
    expect(midLabel.style.transform).not.toBe(touchingTransform);
  });

  it("renders flow chevrons matching the edge stroke color", () => {
    renderEdge(
      <CustomEdge
        id="edge-long"
        source="node-a"
        target="node-b"
        sourceX={520}
        sourceY={120}
        targetX={120}
        targetY={720}
        sourcePosition={"bottom" as never}
        targetPosition={"top" as never}
        selected={false}
        data={{ routeGutterX: 900, contrastStroke: true }}
      />,
    );

    const arrows = screen.getAllByTestId("edge-flow-arrow");
    expect(arrows.length).toBeGreaterThan(0);
    expect(arrows.length).toBeLessThanOrEqual(3);
    expect(arrows[0]).toHaveStyle({ color: TOUCHING_EDGE_STROKE });
    expect(arrows[0].querySelector("path")).toHaveAttribute("fill", "none");
  });
});
