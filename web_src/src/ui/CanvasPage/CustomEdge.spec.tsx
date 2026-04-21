import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

const { setEdges } = vi.hoisted(() => ({
  setEdges: vi.fn(),
}));

vi.mock("@xyflow/react", () => ({
  BaseEdge: ({ className }: { className?: string }) => <div data-testid="base-edge" data-class-name={className} />,
  EdgeLabelRenderer: ({ children }: { children?: ReactNode }) => <>{children}</>,
  getBezierPath: () => ["M0,0 C10,0 20,10 30,10", 15, 5],
  useReactFlow: () => ({
    setEdges,
  }),
}));

import { CustomEdge } from "./CustomEdge";

describe("CustomEdge", () => {
  it("does not show the delete icon or delete on pointer down in live mode", () => {
    const onDelete = vi.fn();

    render(
      <svg>
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
        />
      </svg>,
    );

    expect(screen.queryByTestId("edge-delete-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("edge-delete-hit-area")).not.toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("shows the delete icon and deletes the edge in edit mode", () => {
    const onDelete = vi.fn();

    render(
      <svg>
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
        />
      </svg>,
    );

    expect(screen.getByTestId("edge-delete-icon")).toBeInTheDocument();
    const hitArea = screen.getByTestId("edge-delete-hit-area");

    fireEvent.pointerDown(hitArea, { button: 0, buttons: 1 });
    expect(onDelete).toHaveBeenCalledWith("edge-1");
  });
});
