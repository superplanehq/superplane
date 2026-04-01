import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@xyflow/react", () => ({
  Handle: ({ type, id }: { type: string; id?: string }) => <div data-testid={`handle-${type}-${id || "default"}`} />,
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
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
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
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
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
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
  });
});
