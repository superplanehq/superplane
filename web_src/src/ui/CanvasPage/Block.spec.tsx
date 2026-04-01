import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@xyflow/react", () => ({
  Handle: ({ type, id, className }: { type: string; id?: string; className?: string }) => (
    <div
      data-testid={`handle-${type}-${id || "default"}`}
      data-highlighted={className === "highlighted" ? "true" : "false"}
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
});
