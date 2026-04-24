import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@xyflow/react", () => ({
  Handle: ({
    type,
    id,
    className,
    style,
  }: {
    type: string;
    id?: string;
    className?: string;
    style?: { pointerEvents?: string };
  }) => (
    <div
      data-testid={`handle-${type}-${id || "default"}`}
      data-highlighted={className === "highlighted" ? "true" : "false"}
      data-pointer-events={style?.pointerEvents || "auto"}
    />
  ),
  Position: {
    Left: "left",
    Right: "right",
  },
}));

import { Block, type BlockData } from "./Block";

describe("Block fallback rendering", () => {
  it("renders a fallback node for unknown block types instead of throwing", () => {
    render(
      <Block
        data={{
          label: "Broken Node",
          state: "pending",
          type: "mystery" as unknown as BlockData["type"],
        }}
      />,
    );

    expect(screen.getByText("Broken Node")).toBeInTheDocument();
    expect(screen.getByText("Can't display")).toBeInTheDocument();
  });

  it("renders a fallback node when component props are missing", () => {
    render(
      <Block
        data={{
          label: "Broken Component",
          state: "pending",
          type: "component",
          outputChannels: ["default"],
        }}
      />,
    );

    expect(screen.getByText("Broken Component")).toBeInTheDocument();
    expect(screen.getByText("Can't display")).toBeInTheDocument();
  });

  it("renders a fallback node when trigger props are missing", () => {
    render(
      <Block
        data={{
          label: "Broken Trigger",
          state: "pending",
          type: "trigger",
          outputChannels: ["default"],
        }}
      />,
    );

    expect(screen.getByText("Broken Trigger")).toBeInTheDocument();
    expect(screen.getByText("Can't display")).toBeInTheDocument();
  });

  it("replaces runtime empty states with edit-mode copy", () => {
    render(
      <Block
        canvasMode="edit"
        data={{
          label: "Draft Component",
          state: "pending",
          type: "component",
          outputChannels: ["default"],
          component: {
            title: "Draft Component",
            iconSlug: "box",
            collapsed: false,
            includeEmptyState: true,
            emptyStateProps: {
              title: "Waiting for the first run",
              purpose: "runtime",
            },
          },
        }}
      />,
    );

    expect(screen.getByText("Ready to run...")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for the first run")).not.toBeInTheDocument();
  });

  it("preserves fallback empty states in edit mode", () => {
    render(
      <Block
        canvasMode="edit"
        data={{
          label: "Broken Component",
          state: "pending",
          type: "component",
          outputChannels: ["default"],
        }}
      />,
    );

    expect(screen.getByText("Can't display")).toBeInTheDocument();
    expect(screen.queryByText("Ready to run...")).not.toBeInTheDocument();
  });

  it("does not highlight a right handle when the target node is already connected", () => {
    render(
      <Block
        nodeId="source-node"
        data={{
          label: "Component",
          state: "pending",
          type: "component",
          outputChannels: ["default"],
          component: {
            title: "Component",
            iconSlug: "box",
            collapsed: false,
          },
          _connectingFrom: {
            nodeId: "target-node",
            handleType: "target",
          },
          _allEdges: [
            {
              source: "source-node",
              sourceHandle: "default",
              target: "target-node",
            },
          ],
        }}
      />,
    );

    expect(screen.getByTestId("handle-source-default")).toHaveAttribute("data-highlighted", "false");
  });

  it("disables handle interactivity in live mode", () => {
    render(
      <Block
        canvasMode="live"
        nodeId="component-node"
        data={{
          label: "Component",
          state: "pending",
          type: "component",
          outputChannels: ["default"],
          component: {
            title: "Component",
            iconSlug: "box",
            collapsed: false,
          },
          _allEdges: [
            {
              source: "component-node",
              sourceHandle: "default",
              target: "next-node",
            },
            {
              source: "prev-node",
              sourceHandle: "default",
              target: "component-node",
            },
          ],
        }}
      />,
    );

    expect(screen.getByTestId("handle-target-default")).toHaveAttribute("data-pointer-events", "none");
    expect(screen.getByTestId("handle-source-default")).toHaveAttribute("data-pointer-events", "none");
  });

  it("shows append connector button for end nodes in edit mode", () => {
    const onAppendFromNode = vi.fn();
    const callbacksRef = {
      current: {
        handleNodeClick: vi.fn(),
        onAppendFromNode,
        onNodeEdit: { current: undefined },
        onNodeDelete: { current: undefined },
        onRun: { current: undefined },
        onDuplicate: { current: undefined },
        onDeactivate: { current: undefined },
        onTogglePause: { current: undefined },
        onToggleView: { current: undefined },
        onAnnotationUpdate: { current: undefined },
        onAnnotationBlur: { current: undefined },
        showHeader: true,
        hasMultiSelection: false,
        canvasMode: "edit" as const,
      },
    };

    render(
      <Block
        canvasMode="edit"
        nodeId="end-node"
        data={{
          label: "End node",
          state: "pending",
          type: "component",
          outputChannels: ["default"],
          component: {
            title: "End node",
            iconSlug: "box",
            collapsed: false,
          },
          _callbacksRef: callbacksRef as any,
          _allEdges: [{ source: "prev-node", sourceHandle: "default", target: "end-node" }],
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Add next component" }));
    expect(onAppendFromNode).toHaveBeenCalledWith("end-node", "default");
  });
});
