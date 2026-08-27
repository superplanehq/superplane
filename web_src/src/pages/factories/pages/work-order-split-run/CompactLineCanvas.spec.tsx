import { render, screen } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";

import { CompactLineCanvas } from "./CompactLineCanvas";
import type { SplitRunCanvasModel } from "./splitRunCanvases";

const CREATE_PR_CANVAS: SplitRunCanvasModel = {
  key: "implementation",
  title: "PR Creation",
  nodes: [
    {
      id: "create-pr",
      name: "Create Draft Pull Request",
      component: "github.createPullRequest",
      position: { x: 0, y: 0 },
    },
  ],
  edges: [],
  statuses: { "create-pr": "failed" },
  metrics: { "create-pr": "00:00" },
};

function renderCanvas(selectedId: string | null, nodeEditHref?: (nodeId: string) => string) {
  return render(
    <MemoryRouter>
      <ThemeProvider>
        <ReactFlowProvider>
          <CompactLineCanvas
            canvas={CREATE_PR_CANVAS}
            selectedId={selectedId}
            onSelect={() => undefined}
            showHeader={false}
            nodeEditHref={nodeEditHref}
          />
        </ReactFlowProvider>
      </ThemeProvider>
    </MemoryRouter>,
  );
}

describe("CompactLineCanvas", () => {
  it("draws a thicker ring on the selected node", () => {
    renderCanvas("create-pr");

    const node = screen.getByTestId("split-run-canvas-node-create-pr");
    expect(node).toHaveAttribute("data-selected", "true");
    expect(node.querySelector("[class*='ring-4']")).not.toBeNull();
  });

  it("shows an Edit component control on the selected node", () => {
    renderCanvas("create-pr", (nodeId) => `/edit?node=${nodeId}`);

    const edit = screen.getByRole("link", { name: "Edit component" });
    expect(edit).toHaveAttribute("href", "/edit?node=create-pr");
    expect(edit).toHaveAttribute("data-testid", "split-run-canvas-node-edit");
    expect(edit.className).toContain("top-2");
    expect(edit.className).toContain("right-2");
    expect(edit.className).toContain("bg-foreground");
    expect(edit.className).toContain("text-background");
  });

  it("hides the Edit component control when no node is selected", () => {
    renderCanvas(null, (nodeId) => `/edit?node=${nodeId}`);

    expect(screen.queryByRole("link", { name: "Edit component" })).not.toBeInTheDocument();
  });
});
