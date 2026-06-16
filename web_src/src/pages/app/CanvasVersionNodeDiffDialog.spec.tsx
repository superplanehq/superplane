import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasVersion } from "@/api-client";
import { CanvasVersionNodeDiffDialog, type CanvasVersionNodeDiffContext } from "./CanvasVersionNodeDiffDialog";

function makeVersion(id: string, ownerName = "Alice"): CanvasesCanvasVersion {
  return {
    metadata: {
      id,
      owner: { name: ownerName },
      createdAt: "2026-04-01T00:00:00Z",
    },
    spec: { nodes: [] },
  };
}

function makeContext(): CanvasVersionNodeDiffContext {
  return {
    version: makeVersion("v2"),
    previousVersion: makeVersion("v1"),
  };
}

const noop = () => {};

describe("CanvasVersionNodeDiffDialog", () => {
  it("renders version diff metadata", () => {
    render(<CanvasVersionNodeDiffDialog context={makeContext()} onOpenChange={noop} />);

    expect(screen.getByText("Version Node Diff")).toBeInTheDocument();
    expect(screen.getByText(/Alice/)).toBeInTheDocument();
  });

  it("stays closed without a context", () => {
    render(<CanvasVersionNodeDiffDialog context={null} onOpenChange={noop} />);

    expect(screen.queryByText("Version Node Diff")).not.toBeInTheDocument();
  });
});
